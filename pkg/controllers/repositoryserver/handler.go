// Copyright 2023 The Kanister Authors.
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
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/jpillora/backoff"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/kopia/command/storage"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/poll"
)

type RepoServerHandler struct {
	Req                     ctrl.Request
	Logger                  logr.Logger
	Reconciler              *RepositoryServerReconciler
	KubeCli                 kubernetes.Interface
	RepositoryServer        *crv1alpha1.RepositoryServer
	RepositoryServerSecrets repositoryServerSecrets
}

func (h *RepoServerHandler) CreateOrUpdateOwnedResources(ctx context.Context) error {
	svc, err := h.reconcileService(ctx)
	if err != nil {
		return errors.Wrap(err, "Failed to reconcile service")
	}
	if err = h.getSecretsFromCR(ctx); err != nil {
		return errors.Wrap(err, "Failed to get Kopia API server secrets")
	}
	envVars, pod, err := h.reconcilePod(ctx, svc)
	if err != nil {
		return errors.Wrap(err, "Failed to reconcile Kopia API server pod")
	}
	if err := h.waitForPodReady(ctx, pod); err != nil {
		return errors.Wrap(err, "Kopia API server pod not in ready state")
	}

	// envVars are set only when credentials are of type AWS/Azure.
	// If location credentials are GCP, write them to the pod
	if envVars == nil {
		err = h.writeGCPCredsToPod(ctx, pod)
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *RepoServerHandler) reconcileService(ctx context.Context) (*corev1.Service, error) {
	repoServerNamespace := h.RepositoryServer.Namespace
	serviceName := h.RepositoryServer.Status.ServerInfo.ServiceName
	svc := &corev1.Service{}
	h.Logger.Info("Check if the service resource exists. If exists, reconcile with CR spec")
	err := h.Reconciler.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: repoServerNamespace}, svc)
	if err == nil {
		return h.updateService(ctx, svc)
	}
	if !apierrors.IsNotFound(err) {
		return nil, err
	}
	h.Logger.Info("Service resource not found. Creating new service")
	svc, err = h.createService(ctx, repoServerNamespace)
	if err != nil {
		return nil, err
	}
	h.Logger.Info("Update service name in RepositoryServer /status")
	serverInfo := crv1alpha1.ServerInfo{
		PodName:     h.RepositoryServer.Status.ServerInfo.PodName,
		ServiceName: svc.Name,
	}
	if err := h.updateServerInfoInCRStatus(ctx, serverInfo); err != nil {
		return nil, errors.Wrap(err, "Failed to update service name in RepositoryServer /status")
	}
	return svc, err
}

func (h *RepoServerHandler) updateService(ctx context.Context, svc *corev1.Service) (*corev1.Service, error) {
	svc = h.updateServiceSpec(svc)
	if err := h.Reconciler.Update(ctx, svc); err != nil {
		return nil, err
	}
	return svc, nil
}

func (h *RepoServerHandler) updateServiceSpec(svc *corev1.Service) *corev1.Service {
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

func (h *RepoServerHandler) createService(ctx context.Context, repoServerNamespace string) (*corev1.Service, error) {
	svc := getRepoServerService(repoServerNamespace)
	h.Logger.Info("Set controller reference on the service to allow reconciliation using this controller")
	if err := controllerutil.SetControllerReference(h.RepositoryServer, &svc, h.Reconciler.Scheme); err != nil {
		return nil, err
	}
	err := h.Reconciler.Create(ctx, &svc)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create RepositoryServer service")
	}

	err = poll.WaitWithBackoff(ctx, backoff.Backoff{
		Factor: 2,
		Jitter: false,
		Min:    100 * time.Millisecond,
		Max:    15 * time.Second,
	}, func(ctx context.Context) (bool, error) {
		endpt := corev1.Endpoints{}
		err := h.Reconciler.Get(ctx, types.NamespacedName{Name: svc.Name, Namespace: repoServerNamespace}, &endpt)
		switch {
		case apierrors.IsNotFound(err):
			return false, nil
		case err != nil:
			return false, err
		}
		return true, nil
	})
	return &svc, err
}

func (h *RepoServerHandler) reconcilePod(ctx context.Context, svc *corev1.Service) ([]corev1.EnvVar, *corev1.Pod, error) {
	repoServerNamespace := h.RepositoryServer.Namespace
	podName := h.RepositoryServer.Status.ServerInfo.PodName
	pod := &corev1.Pod{}
	h.Logger.Info("Check if the pod resource exists. If exists, reconcile with CR spec")
	err := h.Reconciler.Get(ctx, types.NamespacedName{Name: podName, Namespace: repoServerNamespace}, pod)
	if err == nil {
		pod, err = h.updatePod(ctx, pod, svc)
		return nil, pod, err
	}
	if !apierrors.IsNotFound(err) {
		return nil, nil, err
	}
	h.Logger.Info("Pod resource not found. Creating new pod")
	var envVars []corev1.EnvVar
	pod, envVars, err = h.createPod(ctx, repoServerNamespace, svc)
	if err != nil {
		return nil, nil, err
	}
	h.Logger.Info("Update pod name in RepositoryServer /status")
	serverInfo := crv1alpha1.ServerInfo{
		PodName:     pod.Name,
		ServiceName: h.RepositoryServer.Status.ServerInfo.ServiceName,
	}
	if err := h.updateServerInfoInCRStatus(ctx, serverInfo); err != nil {
		return nil, nil, errors.Wrap(err, "Failed to update pod name in RepositoryServer /status")
	}

	return envVars, pod, nil
}

func (h *RepoServerHandler) updatePod(ctx context.Context, pod *corev1.Pod, svc *corev1.Service) (*corev1.Pod, error) {
	pod = h.updateServiceNameInPodLabels(pod, svc)
	// TODO: Override the updated pod spec with expected pod spec here
	//  using the data from all Secrets in CR as either EnvVars or Volume Mounts
	// 	before updating it below
	if err := h.Reconciler.Update(ctx, pod); err != nil {
		return nil, err
	}
	return pod, nil
}

func (h *RepoServerHandler) updateServiceNameInPodLabels(pod *corev1.Pod, svc *corev1.Service) *corev1.Pod {
	h.Logger.Info("Check if current service name matches in pod labels")
	if pod.ObjectMeta.Labels[repoServerServiceNameKey] == svc.Name {
		h.Logger.Info("Skipping pod label update. Current service name matches with the pod labels")
		return pod
	}
	h.Logger.Info("Current service name does not match pod labels. Update pod with new service name")
	currentLabel := map[string]string{repoServerServiceNameKey: svc.Name}
	pod.ObjectMeta.Labels = currentLabel
	return pod
}

func (h *RepoServerHandler) createPod(ctx context.Context, repoServerNamespace string, svc *corev1.Service) (*corev1.Pod, []corev1.EnvVar, error) {
	podOverride, err := h.preparePodOverride(ctx)
	if err != nil {
		return nil, nil, err
	}
	podOptions := getPodOptions(repoServerNamespace, podOverride, svc)
	pod, envVars, err := h.setCredDataFromSecretInPod(ctx, podOptions)
	if err != nil {
		return nil, nil, err
	}

	h.Logger.Info("Set controller reference on the pod to allow reconciliation using this controller")
	if err := controllerutil.SetControllerReference(h.RepositoryServer, pod, h.Reconciler.Scheme); err != nil {
		return nil, nil, err
	}
	if err := h.Reconciler.Create(ctx, pod); err != nil {
		return nil, nil, errors.Wrap(err, "Failed to create RepositoryServer pod")
	}
	return pod, envVars, err
}

func (h *RepoServerHandler) writeGCPCredsToPod(ctx context.Context, pod *corev1.Pod) error {
	h.Logger.Info("Setting credentials data from secret as a file in pod")
	storageCredSecret := h.RepositoryServerSecrets.storageCredentials

	if val, ok := storageCredSecret.Data[googleCloudServiceAccFileName]; ok {
		namespace := h.RepositoryServer.Namespace
		pw := kube.NewPodWriter(h.KubeCli, consts.GoogleCloudCredsFilePath, bytes.NewBufferString(string(val)))
		if err := pw.Write(ctx, namespace, pod.Name, repoServerPodContainerName); err != nil {
			return err
		}
	}
	return nil
}

func (h *RepoServerHandler) setCredDataFromSecretInPod(ctx context.Context, podOptions *kube.PodOptions) (*corev1.Pod, []corev1.EnvVar, error) {
	storageCredSecret := h.RepositoryServerSecrets.storageCredentials
	envVars, err := storage.GenerateEnvSpecFromCredentialSecret(storageCredSecret, time.Duration(time.Now().Second()))
	if err != nil {
		return nil, nil, err
	}
	var pod *corev1.Pod
	if envVars != nil {
		h.Logger.Info("Setting credentials data from secret as env variables")
		podOptions.EnvironmentVariables = envVars
	}
	pod, err = kube.GetPodObjectFromPodOptions(h.KubeCli, podOptions)
	if err != nil {
		return nil, nil, err
	}
	return pod, envVars, nil
}

func (h *RepoServerHandler) preparePodOverride(ctx context.Context) (map[string]interface{}, error) {
	namespace := h.RepositoryServer.GetNamespace()
	podOverride, err := getPodOverride(ctx, h.Reconciler, namespace)
	if err != nil {
		return nil, err
	}
	if err := addTLSCertConfigurationInPodOverride(
		&podOverride, h.RepositoryServerSecrets.serverTLS.Name); err != nil {
		return nil, errors.Wrap(err, "Failed to attach TLS Certificate configuration")
	}
	return podOverride, nil
}

func (h *RepoServerHandler) updateServerInfoInCRStatus(ctx context.Context, info crv1alpha1.ServerInfo) error {
	h.Logger.Info("Fetch latest version of RepositoryServer to update the ServerInfo in its status")
	repoServerName := h.RepositoryServer.Name
	repoServerNamespace := h.RepositoryServer.Namespace
	rs := crv1alpha1.RepositoryServer{}
	err := h.Reconciler.Get(ctx, types.NamespacedName{Name: repoServerName, Namespace: repoServerNamespace}, &rs)
	if err != nil {
		return err
	}
	h.Logger.Info("Update the ServerInfo")
	rs.Status.ServerInfo = info
	err = h.Reconciler.Status().Update(ctx, &rs)
	if err != nil {
		return err
	}
	h.Logger.Info("Use this updated RepositoryServer CR")
	h.RepositoryServer = &rs
	return nil
}

func (h *RepoServerHandler) waitForPodReady(ctx context.Context, pod *corev1.Pod) error {
	if err := kube.WaitForPodReady(ctx, h.KubeCli, pod.Namespace, pod.Name); err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed while waiting for pod %s to be ready", pod.Name))
	}
	return nil
}

func (h *RepoServerHandler) updateRepoServerProgress(ctx context.Context, progress crv1alpha1.RepositoryServerProgress) error {
	repoServerName := h.RepositoryServer.Name
	repoServerNamespace := h.RepositoryServer.Namespace
	rs := crv1alpha1.RepositoryServer{}
	err := h.Reconciler.Get(ctx, types.NamespacedName{Name: repoServerName, Namespace: repoServerNamespace}, &rs)
	if err != nil {
		return err
	}
	rs.Status.Progress = progress
	err = h.Reconciler.Status().Update(ctx, &rs)
	if err != nil {
		return err
	}
	h.RepositoryServer = &rs
	return nil
}
