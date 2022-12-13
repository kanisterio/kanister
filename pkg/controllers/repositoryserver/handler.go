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
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/jpillora/backoff"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/kanisterio/kanister/pkg/kopia/command/storage"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/poll"
)

type RepoServerHandler struct {
	Ctx                     context.Context
	Req                     ctrl.Request
	Logger                  logr.Logger
	Reconciler              *RepositoryServerReconciler
	KubeCli                 kubernetes.Interface
	RepositoryServer        *crv1alpha1.RepositoryServer
	RepositoryServerSecrets repositoryServerSecrets
}

func (h *RepoServerHandler) CreateOrUpdateOwnedResources() error {
	svc, err := h.reconcileService()
	if err != nil {
		return err
	}
	if err = h.getSecretsFromCR(); err != nil {
		return err
	}
	pod, err := h.reconcilePod(svc)
	if err != nil {
		return err
	}
	if err := h.waitForPodReady(pod); err != nil {
		return err
	}
	if err := h.createOrConnectKopiaRepository(); err != nil {
		return err
	}

	if err := h.startRepoProxyServer(h.Ctx); err != nil {
		return err
	}

	if err := h.createOrUpdateClientUsers(h.Ctx); err != nil {
		return err
	}
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
		PodName:     h.RepositoryServer.Status.ServerInfo.PodName,
		ServiceName: svc.Name,
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

func (h *RepoServerHandler) reconcilePod(svc *corev1.Service) (*corev1.Pod, error) {
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
	pod, err = h.createPod(repoServerNamespace, svc)
	if err != nil {
		return nil, err
	}
	h.Logger.Info("Update podName in RepositoryServer /status")
	serverInfo := crv1alpha1.ServerInfo{
		PodName:     pod.Name,
		ServiceName: h.RepositoryServer.Status.ServerInfo.ServiceName,
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
	// TODO: Override the updated pod spec with expected pod spec here
	//  using the data from all Secrets in CR as either EnvVars or Volume Mounts
	// 	before updating it below
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

func (h *RepoServerHandler) createPod(repoServerNamespace string, svc *corev1.Service) (*corev1.Pod, error) {
	podOverride, err := h.preparePodOverride()
	if err != nil {
		return nil, err
	}
	podOptions := getPodOptions(repoServerNamespace, podOverride, svc)
	pod, err := h.setCredDataFromSecretInPod(podOptions)
	if err != nil {
		return nil, err
	}
	h.Logger.Info("Set controller reference on Pod to allow reconciliation using this controller")
	if err := controllerutil.SetControllerReference(h.RepositoryServer, pod, h.Reconciler.Scheme); err != nil {
		return nil, err
	}
	if err := h.Reconciler.Create(h.Ctx, pod); err != nil {
		return nil, errors.Wrap(err, "Failed to create RepositoryServer pod")
	}
	return pod, nil
}

func (h *RepoServerHandler) setCredDataFromSecretInPod(podOptions *kube.PodOptions) (*corev1.Pod, error) {
	h.Logger.Info("Setting credentials data from secret as either env variables or files in pod")
	namespace := h.RepositoryServer.Namespace
	storageCredSecret := h.RepositoryServerSecrets.storageCredentials
	envVars, err := storage.GenerateEnvSpecFromCredentialSecret(&storageCredSecret, time.Duration(time.Now().Second()))
	if err != nil {
		return nil, err
	}
	var pod *corev1.Pod
	if envVars != nil {
		podOptions.EnvironmentVariables = envVars
	}
	pod, err = kube.GetPodObjectFromPodOptions(h.KubeCli, podOptions)
	if err != nil {
		return nil, err
	}
	if envVars == nil {
		if val, ok := storageCredSecret.Data[googleCloudServiceAccFileName]; ok {
			val, err = base64.StdEncoding.DecodeString(string(val))
			gcloudCredsFilePath := fmt.Sprintf("%s/%s", googleCloudCredsDirPath, googleCloudServiceAccFileName)
			pw := kube.NewPodWriter(h.KubeCli, gcloudCredsFilePath, bytes.NewBufferString(string(val)))
			if err := pw.Write(h.Ctx, namespace, pod.Name, repoServerPodContainerName); err != nil {
				return nil, err
			}
		}
	}
	return pod, nil
}

func (h *RepoServerHandler) preparePodOverride() (map[string]interface{}, error) {
	namespace := h.RepositoryServer.GetNamespace()
	podOverride, err := getPodOverride(h.Ctx, h.Reconciler, namespace)
	if err != nil {
		return nil, err
	}
	if err := addTLSCertConfigurationInPodOverride(
		&podOverride, h.RepositoryServerSecrets.serverTLS.Name); err != nil {
		return nil, errors.Wrap(err, "Failed to attach TLS Certificate configuration")
	}
	return podOverride, nil
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
