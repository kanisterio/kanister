// Copyright 2022 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package repositoryserver

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/jpillora/backoff"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/poll"
)

type RepoServerHandler struct {
	Ctx              context.Context
	Req              ctrl.Request
	Logger           logr.Logger
	Reconciler       *RepositoryServerReconciler
	KubeCli          kubernetes.Interface
	RepositoryServer *crv1alpha1.RepositoryServer
	OwnerReference   metav1.OwnerReference
}

func (h *RepoServerHandler) Run() error {
	svc, err := h.reconcileService()
	if err != nil {
		return err
	}
	if err := h.reconcileNetworkPolicy(svc); err != nil {
		return err
	}
	podOverride, err := h.preparePodOverride()
	if err != nil {
		return err
	}
	pod, err := h.reconcilePod(podOverride, svc)
	if err != nil {
		return err
	}
	if err := h.waitForPodReady(pod); err != nil {
		return err
	}
	h.connectToRepository()
	h.startRepoProxyServer()
	h.waitForServerToStart()
	h.addClientUsersToServer()
	h.refreshServer()
	return nil
}

func (h *RepoServerHandler) reconcileService() (*corev1.Service, error) {
	repoServerNamespace := h.RepositoryServer.Namespace
	serviceName := h.RepositoryServer.Status.ServerInfo.ServiceName
	svc := &corev1.Service{}
	h.Logger.Info("Check if RepositoryServer Service resource exists. Return if exists....")
	err := h.Reconciler.Get(h.Ctx, types.NamespacedName{Name: serviceName, Namespace: repoServerNamespace}, svc)
	if err == nil {
		return svc, nil
	}
	if !apierrors.IsNotFound(err) {
		return nil, err
	}
	h.Logger.Info("RepositoryServer Service resource not found. Creating new service....")
	svc, err = h.createService(repoServerNamespace)
	if err != nil {
		return nil, err
	}
	h.Logger.Info("Update the ServerInfo in RepositoryServer CR status with serviceName....")
	serverInfo := crv1alpha1.ServerInfo{
		PodName:           h.RepositoryServer.Status.ServerInfo.PodName,
		ServiceName:       svc.Name,
		NetworkPolicyName: h.RepositoryServer.Status.ServerInfo.NetworkPolicyName,
	}
	if err := h.updateServerInfoInCRStatus(serverInfo); err != nil {
		return nil, errors.Wrap(err, "Failed to update ServiceName in RepositoryServer CR status subresource")
	}
	return svc, err
}

func (h *RepoServerHandler) reconcileNetworkPolicy(svc *corev1.Service) error {
	repoServerNamespace := h.RepositoryServer.Namespace
	networkPolicyName := h.RepositoryServer.Status.ServerInfo.NetworkPolicyName
	np := &networkingv1.NetworkPolicy{}
	h.Logger.Info("Check if RepositoryServer NetworkPolicy resource exists. Update it's labels if exists....")
	err := h.Reconciler.Get(h.Ctx, types.NamespacedName{Name: networkPolicyName, Namespace: repoServerNamespace}, np)
	if err == nil {
		return h.updateLabelsInNetworkPolicy(np, svc)
	}
	if !apierrors.IsNotFound(err) {
		return err
	}
	h.Logger.Info("RepositoryServer NetworkPolicy resource not found. Creating new networkPolicy....")
	np, err = h.createNetworkPolicy(repoServerNamespace, svc)
	if err != nil {
		return err
	}
	h.Logger.Info("Update the ServerInfo in RepositoryServer CR status with networkPolicyName....")
	serverInfo := crv1alpha1.ServerInfo{
		PodName:           h.RepositoryServer.Status.ServerInfo.PodName,
		ServiceName:       h.RepositoryServer.Status.ServerInfo.ServiceName,
		NetworkPolicyName: np.Name,
	}
	if err := h.updateServerInfoInCRStatus(serverInfo); err != nil {
		return errors.Wrap(err, "Failed to update networkPolicyName in RepositoryServer CR status subresource")
	}
	return nil
}

func (h *RepoServerHandler) preparePodOverride() (map[string]interface{}, error) {
	namespace := h.RepositoryServer.GetNamespace()
	podOverride, err := getPodOverride(h.Ctx, h.Reconciler, namespace)
	if err != nil {
		return nil, err
	}
	if err := addTLSCertConfigurationInPodOverride(
		&podOverride, h.RepositoryServer.Spec.Server.TLSSecretRef.Name); err != nil {
		return nil, errors.Wrap(err, "Failed to attach TLS Certificate configuration")
	}
	return podOverride, nil
}

func (h *RepoServerHandler) reconcilePod(podOverride map[string]interface{}, svc *corev1.Service) (*corev1.Pod, error) {
	repoServerNamespace := h.RepositoryServer.Namespace
	podName := h.RepositoryServer.Status.ServerInfo.PodName
	pod := &corev1.Pod{}
	h.Logger.Info("Check if RepositoryServer Pod resource exists. Update it's labels if exists....")
	err := h.Reconciler.Get(h.Ctx, types.NamespacedName{Name: podName, Namespace: repoServerNamespace}, pod)
	if err == nil {
		return h.updateLabelsInPod(pod, svc)
	}
	if !apierrors.IsNotFound(err) {
		return nil, err
	}
	h.Logger.Info("RepositoryServer Pod resource not found. Creating new Pod....")
	pod, err = h.createPod(repoServerNamespace, svc, podOverride)
	if err != nil {
		return nil, err
	}
	h.Logger.Info("Update the ServerInfo in RepositoryServer CR status with podName....")
	serverInfo := crv1alpha1.ServerInfo{
		PodName:           pod.Name,
		ServiceName:       h.RepositoryServer.Status.ServerInfo.ServiceName,
		NetworkPolicyName: h.RepositoryServer.Status.ServerInfo.NetworkPolicyName,
	}
	if err := h.updateServerInfoInCRStatus(serverInfo); err != nil {
		return nil, errors.Wrap(err, "Failed to update podName in RepositoryServer CR status subresource")
	}
	return pod, nil
}

func (h *RepoServerHandler) waitForPodReady(pod *corev1.Pod) error {
	if err := kube.WaitForPodReady(h.Ctx, h.KubeCli, pod.Namespace, pod.Name); err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed while waiting for Pod %s to be ready", pod.Name))
	}
	return nil
}

func (h *RepoServerHandler) createService(repoServerNamespace string) (*corev1.Service, error) {
	svc := repoServerServiceResource(repoServerNamespace, h.OwnerReference)
	err := h.Reconciler.Create(h.Ctx, &svc)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create RepositoryServer service")
	}
	err = poll.WaitWithBackoff(h.Ctx, backoff.Backoff{
		Factor: 2,
		Jitter: false,
		Min:    100 * time.Millisecond,
		Max:    15 * time.Second,
	}, func(ctx context.Context) (bool, error) {
		endpt := corev1.Endpoints{}
		err := h.Reconciler.Get(h.Ctx, types.NamespacedName{Name: svc.Name, Namespace: repoServerNamespace}, &endpt)
		switch {
		case apierrors.IsNotFound(err):
			return false, nil
		case err != nil:
			return false, err
		}
		return true, nil
	})
	return &svc, nil
}

func (h *RepoServerHandler) createNetworkPolicy(
	repoServerNamespace string,
	svc *corev1.Service) (*networkingv1.NetworkPolicy, error) {
	podSelector := h.RepositoryServer.Spec.Server.NetworkPolicyIngressRule.PodSelector
	namespaceSelector := h.RepositoryServer.Spec.Server.NetworkPolicyIngressRule.NamespaceSelector
	np := repoServerNetworkPolicy(repoServerNamespace, svc, h.OwnerReference, podSelector, namespaceSelector)
	err := h.Reconciler.Create(h.Ctx, np)
	if err != nil {
		return nil, err
	}
	return np, nil
}

func (h *RepoServerHandler) updateLabelsInNetworkPolicy(np *networkingv1.NetworkPolicy, svc *corev1.Service) error {
	h.Logger.Info("Check if current svcName is present in NP labels and podSelectors....")
	if np.ObjectMeta.Labels[repoServerServiceNameKey] == svc.Name &&
		np.Spec.PodSelector.MatchLabels[repoServerServiceNameKey] == svc.Name {
		h.Logger.Info("Current svcName is present in NP labels and podSelectors. Skipping Label update....")
		return nil
	}
	h.Logger.Info("Update NP with current svcName only if svcName does not match in labels....")
	currentLabel := map[string]string{repoServerServiceNameKey: svc.Name}
	np.ObjectMeta.Labels = currentLabel
	np.Spec.PodSelector.MatchLabels = currentLabel
	if err := h.Reconciler.Update(h.Ctx, np); err != nil {
		return err
	}
	return nil
}

func (h *RepoServerHandler) createPod(
	repoServerNamespace string,
	svc *corev1.Service,
	podOverride map[string]interface{}) (*corev1.Pod, error) {
	podOptions := getPodOptions(repoServerNamespace, podOverride, svc, h.OwnerReference)
	ctx, cancel := context.WithCancel(h.Ctx)
	defer cancel()
	pod, err := kube.CreatePod(ctx, h.KubeCli, podOptions)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create RepositoryServer pod")
	}
	return pod, nil
}

func (h *RepoServerHandler) updateLabelsInPod(pod *corev1.Pod, svc *corev1.Service) (*corev1.Pod, error) {
	h.Logger.Info("Check if current svcName is present in Pod labels....")
	if pod.ObjectMeta.Labels[repoServerServiceNameKey] == svc.Name {
		h.Logger.Info("Current svcName is present in Pod labels. Skipping Label update....")
		return pod, nil
	}
	h.Logger.Info("Update Pod with current svcName only if svcName does not match in labels....")
	currentLabel := map[string]string{repoServerServiceNameKey: svc.Name}
	pod.ObjectMeta.Labels = currentLabel
	if err := h.Reconciler.Update(h.Ctx, pod); err != nil {
		return nil, err
	}
	return pod, nil
}

func (h *RepoServerHandler) connectToRepository() {

}

func (h *RepoServerHandler) startRepoProxyServer() {

}

func (h *RepoServerHandler) waitForServerToStart() {

}

func (h *RepoServerHandler) addClientUsersToServer() {

}

func (h *RepoServerHandler) refreshServer() {

}

func (h *RepoServerHandler) updateServerInfoInCRStatus(info crv1alpha1.ServerInfo) error {
	h.Logger.Info("Fetch latest version of RepositoryServer to update the ServerInfo in it's status")
	repoServerName := h.RepositoryServer.Name
	repoServerNamespace := h.RepositoryServer.Namespace
	rs := crv1alpha1.RepositoryServer{}
	err := h.Reconciler.Get(h.Ctx, types.NamespacedName{Name: repoServerName, Namespace: repoServerNamespace}, &rs)
	if err != nil {
		return err
	}
	h.Logger.Info("Update the ServerInfo")
	rs.Status.ServerInfo = info
	err = h.Reconciler.Status().Update(h.Ctx, &rs)
	if err != nil {
		return err
	}
	h.Logger.Info("Use this updated RepositoryServer CR")
	h.RepositoryServer = &rs
	return nil
}
