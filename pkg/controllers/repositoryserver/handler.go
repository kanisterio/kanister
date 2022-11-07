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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

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
}

func (h *RepoServerHandler) CreateOrUpdateOwnedResources() error {
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
	h.createOrUpdateClientUsers()
	return nil
}

func (h *RepoServerHandler) reconcileService() (*corev1.Service, error) {
	repoServerNamespace := h.RepositoryServer.Namespace
	serviceName := h.RepositoryServer.Status.ServerInfo.ServiceName
	svc := &corev1.Service{}
	h.Logger.Info("Check if Service resource exists. If exists, reconcile with CR spec")
	err := h.Reconciler.Get(h.Ctx, types.NamespacedName{Name: serviceName, Namespace: repoServerNamespace}, svc)
	if err == nil {
		return h.updateService(svc)
	}
	if !apierrors.IsNotFound(err) {
		return nil, err
	}
	h.Logger.Info("Service resource not found. Creating new service")
	svc, err = h.createService(repoServerNamespace)
	if err != nil {
		return nil, err
	}
	h.Logger.Info("Update serviceName in RepositoryServer /status")
	serverInfo := crv1alpha1.ServerInfo{
		PodName:           h.RepositoryServer.Status.ServerInfo.PodName,
		ServiceName:       svc.Name,
		NetworkPolicyName: h.RepositoryServer.Status.ServerInfo.NetworkPolicyName,
	}
	if err := h.updateServerInfoInCRStatus(serverInfo); err != nil {
		return nil, errors.Wrap(err, "Failed to update ServiceName in RepositoryServer /status")
	}
	return svc, err
}

func (h *RepoServerHandler) updateService(svc *corev1.Service) (*corev1.Service, error) {
	svc = h.updateServiceSpecInService(svc)
	if err := h.Reconciler.Update(h.Ctx, svc); err != nil {
		return nil, err
	}
	return svc, nil
}

func (h *RepoServerHandler) updateServiceSpecInService(svc *corev1.Service) *corev1.Service {
	serviceSpec := corev1.ServiceSpec{
		Ports: []corev1.ServicePort{
			{
				Name:     fmt.Sprintf("%s-port", repoServerService),
				Protocol: repoServerServiceProtocol,
				Port:     repoServerServicePort,
			},
		},
		Selector: map[string]string{repoServerServiceNameKey: svc.Name},
	}
	svc.Spec = serviceSpec
	return svc
}

func (h *RepoServerHandler) createService(repoServerNamespace string) (*corev1.Service, error) {
	svc := getRepoServerService(repoServerNamespace)
	h.Logger.Info("Set controller reference on Service to allow reconciliation using this controller")
	if err := controllerutil.SetControllerReference(h.RepositoryServer, &svc, h.Reconciler.Scheme); err != nil {
		return nil, err
	}
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

func (h *RepoServerHandler) reconcileNetworkPolicy(svc *corev1.Service) error {
	repoServerNamespace := h.RepositoryServer.Namespace
	networkPolicyName := h.RepositoryServer.Status.ServerInfo.NetworkPolicyName
	np := &networkingv1.NetworkPolicy{}
	h.Logger.Info("Check if NetworkPolicy resource exists. If exists, reconcile with CR spec")
	err := h.Reconciler.Get(h.Ctx, types.NamespacedName{Name: networkPolicyName, Namespace: repoServerNamespace}, np)
	if err == nil {
		return h.updateNetworkPolicy(np, svc)
	}
	if !apierrors.IsNotFound(err) {
		return err
	}
	h.Logger.Info("NetworkPolicy resource not found. Creating new networkPolicy")
	np, err = h.createNetworkPolicy(repoServerNamespace, svc)
	if err != nil {
		return err
	}
	h.Logger.Info("Update networkPolicyName in RepositoryServer /status")
	serverInfo := crv1alpha1.ServerInfo{
		PodName:           h.RepositoryServer.Status.ServerInfo.PodName,
		ServiceName:       h.RepositoryServer.Status.ServerInfo.ServiceName,
		NetworkPolicyName: np.Name,
	}
	if err := h.updateServerInfoInCRStatus(serverInfo); err != nil {
		return errors.Wrap(err, "Failed to update networkPolicyName in RepositoryServer /status")
	}
	return nil
}

func (h *RepoServerHandler) updateNetworkPolicy(np *networkingv1.NetworkPolicy, svc *corev1.Service) error {
	np, err := h.updateServiceNameInNetworkPolicyLabels(np, svc)
	if err != nil {
		return err
	}
	np = h.updateIngressRuleInNetworkPolicy(np)
	if err := h.Reconciler.Update(h.Ctx, np); err != nil {
		return err
	}
	return nil
}

func (h *RepoServerHandler) updateServiceNameInNetworkPolicyLabels(
	np *networkingv1.NetworkPolicy,
	svc *corev1.Service) (*networkingv1.NetworkPolicy, error) {
	h.Logger.Info("Check if current svcName matches in NP labels and podSelectors")
	if np.ObjectMeta.Labels[repoServerServiceNameKey] == svc.Name &&
		np.Spec.PodSelector.MatchLabels[repoServerServiceNameKey] == svc.Name {
		h.Logger.Info("Skipping NP Label update. Current svcName matches with NP labels and podSelectors")
		return np, nil
	}
	h.Logger.Info("Current svcName does not match in NP labels. Update NP with new svcName")
	currentLabel := map[string]string{repoServerServiceNameKey: svc.Name}
	np.ObjectMeta.Labels = currentLabel
	np.Spec.PodSelector.MatchLabels = currentLabel
	return np, nil
}

func (h *RepoServerHandler) updateIngressRuleInNetworkPolicy(
	np *networkingv1.NetworkPolicy) *networkingv1.NetworkPolicy {
	protocolTCP := corev1.ProtocolTCP
	port := intstr.FromInt(repoServerServicePort)
	ingressRule := []networkingv1.NetworkPolicyIngressRule{
		{
			From: []networkingv1.NetworkPolicyPeer{
				{
					PodSelector:       h.RepositoryServer.Spec.Server.NetworkPolicyIngressRule.PodSelector,
					NamespaceSelector: h.RepositoryServer.Spec.Server.NetworkPolicyIngressRule.NamespaceSelector,
				},
			},
			Ports: []networkingv1.NetworkPolicyPort{
				{
					Protocol: &protocolTCP,
					Port:     &port,
				},
			},
		},
	}
	np.Spec.Ingress = ingressRule
	return np
}

func (h *RepoServerHandler) createNetworkPolicy(
	repoServerNamespace string,
	svc *corev1.Service) (*networkingv1.NetworkPolicy, error) {
	podSelector := h.RepositoryServer.Spec.Server.NetworkPolicyIngressRule.PodSelector
	namespaceSelector := h.RepositoryServer.Spec.Server.NetworkPolicyIngressRule.NamespaceSelector
	np := getRepoServerNetworkPolicy(repoServerNamespace, svc, podSelector, namespaceSelector)
	h.Logger.Info("Set controller reference on NetworkPolicy to allow reconciliation using this controller")
	if err := controllerutil.SetControllerReference(h.RepositoryServer, np, h.Reconciler.Scheme); err != nil {
		return nil, err
	}
	err := h.Reconciler.Create(h.Ctx, np)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create RepositoryServer networkPolicy")
	}
	return np, nil
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
	h.Logger.Info("Check if Pod resource exists. If exists, reconcile with CR spec")
	err := h.Reconciler.Get(h.Ctx, types.NamespacedName{Name: podName, Namespace: repoServerNamespace}, pod)
	if err == nil {
		return h.updatePod(pod, svc)
	}
	if !apierrors.IsNotFound(err) {
		return nil, err
	}
	h.Logger.Info("Pod resource not found. Creating new Pod")
	pod, err = h.createPod(repoServerNamespace, svc, podOverride)
	if err != nil {
		return nil, err
	}
	h.Logger.Info("Update podName in RepositoryServer /status")
	serverInfo := crv1alpha1.ServerInfo{
		PodName:           pod.Name,
		ServiceName:       h.RepositoryServer.Status.ServerInfo.ServiceName,
		NetworkPolicyName: h.RepositoryServer.Status.ServerInfo.NetworkPolicyName,
	}
	if err := h.updateServerInfoInCRStatus(serverInfo); err != nil {
		return nil, errors.Wrap(err, "Failed to update podName in RepositoryServer /status")
	}
	return pod, nil
}

func (h *RepoServerHandler) updatePod(pod *corev1.Pod, svc *corev1.Service) (*corev1.Pod, error) {
	pod, err := h.updateServiceNameInPodLabels(pod, svc)
	if err != nil {
		return nil, err
	}
	// TODO: Reconcile all SecretRefs in CR with Secrets Mounts in Pod here, before updating the pod below
	if err := h.Reconciler.Update(h.Ctx, pod); err != nil {
		return nil, err
	}
	return pod, nil
}

func (h *RepoServerHandler) updateServiceNameInPodLabels(pod *corev1.Pod, svc *corev1.Service) (*corev1.Pod, error) {
	h.Logger.Info("Check if current svcName matches in Pod labels")
	if pod.ObjectMeta.Labels[repoServerServiceNameKey] == svc.Name {
		h.Logger.Info("Skipping Pod Label update. Current svcName matches with Pod labels")
		return pod, nil
	}
	h.Logger.Info("Current svcName does not match in Pod labels. Update Pod with new svcName")
	currentLabel := map[string]string{repoServerServiceNameKey: svc.Name}
	pod.ObjectMeta.Labels = currentLabel
	return pod, nil
}

func (h *RepoServerHandler) createPod(
	repoServerNamespace string,
	svc *corev1.Service,
	podOverride map[string]interface{}) (*corev1.Pod, error) {
	podOptions := getPodOptions(repoServerNamespace, podOverride, svc)
	pod, ns, err := kube.GetPodObjectFromPodOptions(h.KubeCli, podOptions)
	if err != nil {
		return nil, err
	}
	pod.ObjectMeta.Namespace = ns
	h.Logger.Info("Set controller reference on Pod to allow reconciliation using this controller")
	if err := controllerutil.SetControllerReference(h.RepositoryServer, pod, h.Reconciler.Scheme); err != nil {
		return nil, err
	}
	if err := h.Reconciler.Create(h.Ctx, pod); err != nil {
		return nil, errors.Wrap(err, "Failed to create RepositoryServer pod")
	}
	return pod, nil
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

func (h *RepoServerHandler) waitForPodReady(pod *corev1.Pod) error {
	if err := kube.WaitForPodReady(h.Ctx, h.KubeCli, pod.Namespace, pod.Name); err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed while waiting for Pod %s to be ready", pod.Name))
	}
	return nil
}

func (h *RepoServerHandler) connectToRepository() {

}

func (h *RepoServerHandler) startRepoProxyServer() {

}

func (h *RepoServerHandler) waitForServerToStart() {

}

func (h *RepoServerHandler) createOrUpdateClientUsers() {
	h.refreshServer()
}

func (h *RepoServerHandler) refreshServer() {

}
