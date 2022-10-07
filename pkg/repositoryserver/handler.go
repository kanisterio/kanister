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

	"github.com/jpillora/backoff"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/client/clientset/versioned"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/poll"
)

type Handler struct {
	Ctx              context.Context
	KubeCli          kubernetes.Interface
	CrCli            versioned.Interface
	RepositoryServer *crv1alpha1.RepositoryServer
	OwnerReference   metav1.OwnerReference
}

func (h *Handler) RunRepositoryProxyServer() error {
	svc, err := h.createService()
	if err != nil {
		return err
	}
	if err := h.createNetworkPolicy(svc); err != nil {
		return err
	}
	podOverride, err := h.preparePodOverride()
	if err != nil {
		return err
	}
	pod, err := h.createRepoServerPod(podOverride, svc)
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

func (h *Handler) createService() (*corev1.Service, error) {
	namespace := h.RepositoryServer.GetNamespace()
	svc := repoServerPodService(namespace, h.OwnerReference)
	svc, err := h.KubeCli.CoreV1().Services(namespace).Create(h.Ctx, svc, metav1.CreateOptions{})
	if err != nil {
		return svc, errors.Wrap(err, "Failed to create RepositoryServer service")
	}
	err = poll.WaitWithBackoff(h.Ctx, backoff.Backoff{
		Factor: 2,
		Jitter: false,
		Min:    100 * time.Millisecond,
		Max:    15 * time.Second,
	}, func(ctx context.Context) (bool, error) {
		_, err := h.KubeCli.CoreV1().Endpoints(namespace).Get(ctx, svc.Name, metav1.GetOptions{})
		switch {
		case apierrors.IsNotFound(err):
			return false, nil
		case err != nil:
			return false, err
		}
		return true, nil
	})
	serverInfo := crv1alpha1.ServerInfo{
		PodName:           h.RepositoryServer.Status.ServerInfo.PodName,
		ServiceName:       svc.Name,
		NetworkPolicyName: h.RepositoryServer.Status.ServerInfo.NetworkPolicyName,
	}
	if err := h.updateServerInfoInCR(serverInfo); err != nil {
		return nil, errors.Wrap(err, "Failed to update ServiceName in RepositoryServer CR status subresource")
	}
	return svc, err
}

func (h *Handler) createNetworkPolicy(svc *corev1.Service) error {
	namespace := h.RepositoryServer.GetNamespace()
	podSelector := h.RepositoryServer.Spec.Server.NetworkPolicyIngressRule.PodSelector
	namespaceSelector := h.RepositoryServer.Spec.Server.NetworkPolicyIngressRule.NamespaceSelector
	np := repoServerNetworkPolicy(namespace, svc, h.OwnerReference, podSelector, namespaceSelector)
	np, err := h.KubeCli.NetworkingV1().NetworkPolicies(namespace).Create(h.Ctx, np, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	serverInfo := crv1alpha1.ServerInfo{
		PodName:           h.RepositoryServer.Status.ServerInfo.PodName,
		ServiceName:       h.RepositoryServer.Status.ServerInfo.ServiceName,
		NetworkPolicyName: np.Name,
	}
	if err := h.updateServerInfoInCR(serverInfo); err != nil {
		return errors.Wrap(err, "Failed to update networkPolicyName in RepositoryServer CR status subresource")
	}
	return nil
}

func (h *Handler) preparePodOverride() (map[string]interface{}, error) {
	namespace := h.RepositoryServer.GetNamespace()
	podOverride := getPodOverride(h.Ctx, h.KubeCli, namespace)
	if err := addTLSCertConfigurationInPodOverride(
		&podOverride, h.RepositoryServer.Spec.Server.TLSSecretRef.Name); err != nil {
		return nil, errors.Wrap(err, "Failed to attach TLS Certificate configuration")
	}
	return podOverride, nil
}

func (h *Handler) createRepoServerPod(
	podOverride map[string]interface{},
	svc *corev1.Service) (*corev1.Pod, error) {
	namespace := h.RepositoryServer.GetNamespace()
	podOptions := getPodOptions(namespace, podOverride, svc, h.OwnerReference)
	ctx, cancel := context.WithCancel(h.Ctx)
	defer cancel()
	pod, err := kube.CreatePod(ctx, h.KubeCli, podOptions)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create RepositoryServer pod")
	}
	serverInfo := crv1alpha1.ServerInfo{
		PodName:           pod.Name,
		ServiceName:       h.RepositoryServer.Status.ServerInfo.ServiceName,
		NetworkPolicyName: h.RepositoryServer.Status.ServerInfo.NetworkPolicyName,
	}
	if err := h.updateServerInfoInCR(serverInfo); err != nil {
		return nil, errors.Wrap(err, "Failed to update podName in RepositoryServer CR status subresource")
	}
	return pod, nil
}

func (h *Handler) waitForPodReady(pod *corev1.Pod) error {
	if err := kube.WaitForPodReady(h.Ctx, h.KubeCli, pod.Namespace, pod.Name); err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed while waiting for Pod %s to be ready", pod.Name))
	}
	return nil
}

func (h *Handler) connectToRepository() {

}

func (h *Handler) startRepoProxyServer() {

}

func (h *Handler) waitForServerToStart() {

}

func (h *Handler) addClientUsersToServer() {

}

func (h *Handler) refreshServer() {

}

func (h *Handler) updateServerInfoInCR(info crv1alpha1.ServerInfo) error {
	// Fetch latest version of RepositoryServer
	repoServerName := h.RepositoryServer.Name
	repoServerNamespace := h.RepositoryServer.Namespace
	rs, err := h.CrCli.CrV1alpha1().RepositoryServers(repoServerNamespace).Get(context.Background(), repoServerName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	// Update the ServerInfo
	rs.Status.ServerInfo = info
	rs, err = h.CrCli.CrV1alpha1().RepositoryServers(repoServerNamespace).UpdateStatus(context.Background(), rs, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	// Use this updated version
	h.RepositoryServer = rs
	return nil
}
