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
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/jpillora/backoff"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/poll"
)

const (
	repoServerService = "repo-server-service"
	repoServerNP      = "repo-server-networkpolicy"
	repoServerPod     = "repo-server-pod"

	repoServerServiceNameKey  = "name"
	repoServerServiceProtocol = "TCP"
	repoServerServicePort     = 51515
	repoServerAddressFormat   = "https://%s:%d"
	repoServerUsernameFormat  = "%s@%s"

	repoServerPodContainerName    = "repo-server-container"
	googleCloudCredsDirPath       = "/mnt/secrets/creds/gcloud"
	googleCloudServiceAccFileName = "service-account.json"

	// CustomCACertName is the name of the custom root CA certificate
	customCACertName        = "custom-ca-bundle.pem"
	tlsCertVolumeName       = "kopia-cert"
	tlsCertDefaultMountPath = "/mnt/secrets/tlscert"
	tlsKeyPath              = "/mnt/secrets/tlscert/tls.key"
	tlsCertPath             = "/mnt/secrets/tlscert/tls.crt"
)

func getRepoServerService(namespace string) corev1.Service {
	name := fmt.Sprintf("%s-%s", repoServerService, rand.String(5))
	return corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{repoServerServiceNameKey: name},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:     fmt.Sprintf("%s-port", repoServerService),
					Protocol: repoServerServiceProtocol,
					Port:     repoServerServicePort,
				},
			},
			Selector: map[string]string{repoServerServiceNameKey: name},
		},
	}
}

// RepoServerNetworkPolicy returns a network policy with appropriate pod selector
func getRepoServerNetworkPolicy(
	namespace string,
	svc *corev1.Service,
	podSelector *metav1.LabelSelector,
	namespaceSelector *metav1.LabelSelector,
) *networkingv1.NetworkPolicy {
	protocolTCP := corev1.ProtocolTCP
	port := intstr.FromInt(repoServerServicePort)
	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", repoServerNP),
			Namespace:    namespace,
			Labels:       map[string]string{repoServerServiceNameKey: svc.Name},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{repoServerServiceNameKey: svc.Name},
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					From: []networkingv1.NetworkPolicyPeer{
						{
							PodSelector:       podSelector,
							NamespaceSelector: namespaceSelector,
						},
					},
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: &protocolTCP,
							Port:     &port,
						},
					},
				},
			},
		},
	}
}

func getPodOverride(ctx context.Context, reconciler *RepositoryServerReconciler, namespace string) (map[string]interface{}, error) {
	podName := os.Getenv("HOSTNAME")
	pod := corev1.Pod{}
	err := reconciler.Get(ctx, types.NamespacedName{Name: podName, Namespace: namespace}, &pod)
	if err != nil {
		return nil, err
	}
	uidguid := int64(0)
	nonRootBool := false
	podOverride := map[string]interface{}{
		"securityContext": corev1.PodSecurityContext{
			RunAsUser:    &uidguid,
			RunAsNonRoot: &nonRootBool,
		},
	}
	if pod.Spec.NodeSelector != nil {
		podOverride["nodeSelector"] = pod.Spec.NodeSelector
	}
	if pod.Spec.Tolerations != nil {
		podOverride["tolerations"] = pod.Spec.Tolerations
	}
	podOverrideSpecForCACertificate(pod.Spec, podOverride)
	return podOverride, nil
}

func podOverrideSpecForCACertificate(podSpec corev1.PodSpec, podOverride map[string]interface{}) {
	if volName, proceed := volumeMountSpecForName(podSpec, podOverride, customCACertName); proceed {
		volumeSpecForName(podSpec, podOverride, volName)
	}
}

// volumeMountSpecForName adds a container spec to the override map
// if the pod spec's volumeMount's SubPath matches with a given certificate name.
// The container spec will include the matching volumeMount.
func volumeMountSpecForName(podSpec corev1.PodSpec, podOverride map[string]interface{}, certName string) (string, bool) {
	if certName == "" {
		return "", false
	}
	for _, ctr := range podSpec.Containers {
		for _, mount := range ctr.VolumeMounts {
			if mount.SubPath != certName {
				continue
			}
			mountList := []corev1.VolumeMount{mount}
			ctr := &corev1.Container{
				Name:         "container",
				VolumeMounts: mountList,
			}
			podOverride["containers"] = []corev1.Container{*ctr}
			return mount.Name, true
		}
	}
	return "", false
}

// volumeSpecForName adds a pod's volume spec to the override map
// if the volume's name matches with a given volumeName
func volumeSpecForName(podSpec corev1.PodSpec, podOverride map[string]interface{}, volumeName string) {
	for _, vol := range podSpec.Volumes {
		if vol.Name == volumeName {
			podOverride["volumes"] = []corev1.Volume{vol}
			return
		}
	}
}

func addTLSCertConfigurationInPodOverride(podOverride *map[string]interface{}, tlsCertSecretName string) error {
	podSpecBytes, err := json.Marshal(*podOverride)
	if err != nil {
		return errors.Wrap(err, "Failed to marshal Pod Override")
	}

	var podOverrideSpec corev1.PodSpec
	if err := json.Unmarshal(podSpecBytes, &podOverrideSpec); err != nil {
		return errors.Wrap(err, "Failed to unmarshal Pod Override Spec")
	}

	podOverrideSpec.Volumes = append(podOverrideSpec.Volumes, corev1.Volume{
		Name: tlsCertVolumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: tlsCertSecretName,
			},
		},
	})

	if len(podOverrideSpec.Containers) == 0 {
		podOverrideSpec.Containers = append(podOverrideSpec.Containers, corev1.Container{
			Name: "container",
		})
	}

	podOverrideSpec.Containers[0].VolumeMounts = append(podOverrideSpec.Containers[0].VolumeMounts, corev1.VolumeMount{
		Name:      tlsCertVolumeName,
		MountPath: tlsCertDefaultMountPath,
	})

	podSpecBytes, err = json.Marshal(podOverrideSpec)
	if err != nil {
		return errors.Wrap(err, "Failed to marshal Pod Override Spec")
	}

	if err := json.Unmarshal(podSpecBytes, podOverride); err != nil {
		return errors.Wrap(err, "Failed to unmarshal Pod Override")
	}

	return nil
}

func getPodOptions(namespace string, podOverride map[string]interface{}, svc *corev1.Service) *kube.PodOptions {
	return &kube.PodOptions{
		Namespace:     namespace,
		GenerateName:  fmt.Sprintf("%s-", repoServerPod),
		Image:         consts.KanisterToolsImage,
		ContainerName: repoServerPodContainerName,
		Command:       []string{"bash", "-c", "tail -f /dev/null"},
		PodOverride:   podOverride,
		Labels:        map[string]string{repoServerServiceNameKey: svc.Name},
	}
}

func getPodAddress(ctx context.Context, cli kubernetes.Interface, namespace, podName string) (string, error) {
	p, err := cli.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrap(err, "Failed to get pod")
	}
	return fmt.Sprintf(repoServerAddressFormat, p.Status.PodIP, repoServerServicePort), nil
}

// WaitTillCommandSucceed returns error if the Command fails to pass without error before default timeout
func WaitTillCommandSucceed(ctx context.Context, cli kubernetes.Interface, cmd []string, namespace, podName, container string) error {
	err := poll.WaitWithBackoff(ctx, backoff.Backoff{
		Factor: 2,
		Jitter: false,
		Min:    100 * time.Millisecond,
		Max:    180 * time.Second,
	}, func(context.Context) (bool, error) {
		stdout, stderr, exErr := kube.Exec(cli, namespace, podName, container, cmd, nil)
		format.Log(podName, container, stdout)
		format.Log(podName, container, stderr)
		if exErr != nil {
			return false, nil
		}
		return true, nil
	})
	return err
}
