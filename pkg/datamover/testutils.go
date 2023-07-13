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

package datamover

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"

	awsconfig "github.com/kanisterio/kanister/pkg/aws"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/controllers/repositoryserver"
	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kopia"
	kopiacmd "github.com/kanisterio/kanister/pkg/kopia/command"
	"github.com/kanisterio/kanister/pkg/kopia/repository"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/testutil"
)

const (
	repositoryServerTestNamespace    = "repository-server-test-namespace-"
	repositoryServerTestPod          = "repository-server-test-pod-"
	repositoryServerTestService      = "repository-server-test-service-"
	kanisterToolsImage               = "ghcr.io/kanisterio/kanister-tools:0.93.0"
	kanisterToolsImageEnvName        = "KANISTER_TOOLS"
	defaultKopiaRepositoryPassword   = "test1234"
	defaultKopiaRepositoryUser       = "repositoryuser"
	defaultKopiaRepositoryPath       = "repository-server-test"
	defaultKopiaRepositoryHost       = "localhost"
	defaultServerStartTimeout        = 10 * time.Minute
	testRepoServerName               = "test-repo-server"
	testKopiaRepoServerAdminUsername = "testadmin@localhost"
	testKopiaRepoServerUsername      = "testuser-"
	testKopiaRepoServerAdminPassword = "testpass1234"
	testTLSKeyPath                   = "/tmp/tls/tls.key"
	testTLSCertPath                  = "/tmp/tls/tls.crt"
)

func createRepositoryServerTestNamespace(ctx context.Context, cli kubernetes.Interface) (*corev1.Namespace, error) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: repositoryServerTestNamespace,
		},
	}
	nsCreated, err := cli.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return nsCreated, nil
}

func createRepositoryServerTestPod(ctx context.Context, cli kubernetes.Interface, namespace string, secret *corev1.Secret) (*corev1.Pod, error) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: repositoryServerTestPod,
			Namespace:    namespace,
			Labels:       map[string]string{"app": "test-repo-server"},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-repository-server-container",
					Image: consts.LatestKanisterToolsImage,
					Ports: []corev1.ContainerPort{
						{
							HostPort:      51515,
							ContainerPort: 51515,
						},
					},
					Env: []corev1.EnvVar{
						{
							Name:  awsconfig.AccessKeyID,
							Value: os.Getenv(testutil.S3CompliantAccessKeyIDEnv),
						},
						{
							Name:  awsconfig.SecretAccessKey,
							Value: os.Getenv(testutil.S3CompliantSecretAccessKeyEnv),
						},
						{
							Name:  awsconfig.Region,
							Value: testutil.TestS3Region,
						},
						{
							Name:  "LOCATION_ENDPOINT",
							Value: os.Getenv(testutil.S3CompliantLocationEndpointEnv),
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "tls-certs",
							MountPath: "/tmp/tls/",
							ReadOnly:  true,
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "tls-certs",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: secret.GetName(),
						},
					},
				},
			},
		},
	}

	podCreated, err := cli.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return podCreated, nil
}

func createRepositoryServerTestService(ctx context.Context, cli kubernetes.Interface, namespace string) (*corev1.Service, error) {
	name := fmt.Sprintf("%s%s", repositoryServerTestService, rand.String(5))

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{"name": name},
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeNodePort,
			Selector: map[string]string{"app": "test-repo-server"},
			Ports: []corev1.ServicePort{
				{
					Name:     "test-repo-server-service-port",
					Protocol: "TCP",
					Port:     51515,
					// Selects Random Port from NodePort Range
					NodePort:   int32(rand.IntnRange(30000, 32767)),
					TargetPort: intstr.FromInt(51515),
				},
			},
		},
	}

	serviceCreated, err := cli.CoreV1().Services(namespace).Create(ctx, service, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	return serviceCreated, nil
}

func createTestKopiaRepository(location *corev1.Secret, cli kubernetes.Interface, namespace string, pod *corev1.Pod) error {
	contentCacheMB, metadataCacheMB := kopiacmd.GetGeneralCacheSizeSettings()
	args := kopiacmd.RepositoryCommandArgs{
		CommandArgs: &kopiacmd.CommandArgs{
			RepoPassword:   defaultKopiaRepositoryPassword,
			ConfigFilePath: kopiacmd.DefaultConfigFilePath,
			LogDirectory:   kopiacmd.DefaultLogDirectory,
		},
		CacheDirectory:  kopiacmd.DefaultCacheDirectory,
		Hostname:        defaultKopiaRepositoryHost,
		ContentCacheMB:  contentCacheMB,
		MetadataCacheMB: metadataCacheMB,
		Username:        defaultKopiaRepositoryUser,
		RepoPathPrefix:  defaultKopiaRepositoryPath,
		Location:        location.Data,
	}

	return repository.ConnectToOrCreateKopiaRepository(
		cli,
		namespace,
		pod.GetName(),
		pod.Spec.Containers[0].Name,
		args,
	)
}

func startTestKopiaRepositoryServer(cli kubernetes.Interface, namespace string, pod *corev1.Pod) error {
	cmd := kopiacmd.ServerStart(
		kopiacmd.ServerStartCommandArgs{
			CommandArgs: &kopiacmd.CommandArgs{
				RepoPassword:   "",
				ConfigFilePath: kopiacmd.DefaultConfigFilePath,
				LogDirectory:   kopiacmd.DefaultLogDirectory,
			},
			ServerAddress:    "https://0.0.0.0:51515",
			TLSCertFile:      testTLSCertPath,
			TLSKeyFile:       testTLSKeyPath,
			ServerUsername:   testKopiaRepoServerAdminUsername,
			ServerPassword:   testKopiaRepoServerAdminPassword,
			AutoGenerateCert: false,
			Background:       true,
		})

	stdout, stderr, err := kube.Exec(cli, namespace, pod.GetName(), pod.Spec.Containers[0].Name, cmd, nil)
	format.Log(pod.GetName(), pod.Spec.Containers[0].Name, stdout)
	format.Log(pod.GetName(), pod.Spec.Containers[0].Name, stderr)
	if err != nil {
		return errors.Wrap(err, "Failed to start Kopia API server")
	}
	return nil
}

func getServerStatusCommand(ctx context.Context, cli kubernetes.Interface, namespace string, tlsSecret *corev1.Secret) ([]string, error) {
	fingerprint, err := kopia.ExtractFingerprintFromCertSecret(ctx, cli, tlsSecret.GetName(), namespace)
	if err != nil {
		return nil, errors.Wrap(err, "Error Extracting Fingerprint from the TLS Certificates")
	}

	cmd := kopiacmd.ServerStatus(
		kopiacmd.ServerStatusCommandArgs{
			CommandArgs: &kopiacmd.CommandArgs{
				RepoPassword:   "",
				ConfigFilePath: kopiacmd.DefaultConfigFilePath,
				LogDirectory:   kopiacmd.DefaultLogDirectory,
			},
			ServerAddress:  "https://0.0.0.0:51515",
			ServerUsername: testKopiaRepoServerAdminUsername,
			ServerPassword: testKopiaRepoServerAdminPassword,
			Fingerprint:    fingerprint,
		})
	return cmd, nil
}

func waitForServerReady(ctx context.Context, cli kubernetes.Interface, namespace string, pod *corev1.Pod, tlsSecret *corev1.Secret) error {
	cmd, err := getServerStatusCommand(ctx, cli, namespace, tlsSecret)
	if err != nil {
		return errors.Wrap(err, "Error Getting Server Status Command")
	}

	ctx, cancel := context.WithTimeout(ctx, defaultServerStartTimeout)
	defer cancel()

	return repositoryserver.WaitTillCommandSucceed(ctx, cli, cmd, namespace, pod.GetName(), pod.Spec.Containers[0].Name)
}

func addTestUserInKopiaRepositoryServer(cli kubernetes.Interface, namespace string, pod *corev1.Pod, username string) error {
	testUser := fmt.Sprintf("%s@%s", username, defaultKopiaRepositoryHost)
	cmd := kopiacmd.ServerAddUser(
		kopiacmd.ServerAddUserCommandArgs{
			CommandArgs: &kopiacmd.CommandArgs{
				RepoPassword:   defaultKopiaRepositoryPassword,
				ConfigFilePath: kopiacmd.DefaultConfigFilePath,
				LogDirectory:   kopiacmd.DefaultLogDirectory,
			},
			NewUsername:  testUser,
			UserPassword: testKopiaRepoServerAdminPassword,
		})
	stdout, stderr, err := kube.Exec(cli, namespace, pod.GetName(), pod.Spec.Containers[0].Name, cmd, nil)
	format.Log(pod.GetName(), pod.Spec.Containers[0].Name, stdout)
	format.Log(pod.GetName(), pod.Spec.Containers[0].Name, stderr)
	if err != nil {
		return errors.Wrap(err, "Failed to add users in Kopia API server")
	}
	return nil
}

func refreshTestKopiaRepositoryServer(ctx context.Context, cli kubernetes.Interface, namespace string, pod *corev1.Pod, tlsSecret *corev1.Secret) error {
	fingerprint, err := kopia.ExtractFingerprintFromCertSecret(ctx, cli, tlsSecret.GetName(), namespace)
	if err != nil {
		return errors.Wrap(err, "Error Extracting Fingerprint from the TLS Certificates")
	}

	cmd := kopiacmd.ServerRefresh(
		kopiacmd.ServerRefreshCommandArgs{
			CommandArgs: &kopiacmd.CommandArgs{
				RepoPassword:   defaultKopiaRepositoryPassword,
				ConfigFilePath: kopiacmd.DefaultConfigFilePath,
				LogDirectory:   kopiacmd.DefaultLogDirectory,
			},
			ServerAddress:  "https://0.0.0.0:51515",
			ServerUsername: testKopiaRepoServerAdminUsername,
			ServerPassword: testKopiaRepoServerAdminPassword,
			Fingerprint:    fingerprint,
		})
	stdout, stderr, err := kube.Exec(cli, namespace, pod.GetName(), pod.Spec.Containers[0].Name, cmd, nil)
	format.Log(pod.GetName(), pod.Spec.Containers[0].Name, stdout)
	format.Log(pod.GetName(), pod.Spec.Containers[0].Name, stderr)
	if err != nil {
		return errors.Wrap(err, "Failed to refresh Kopia API server")
	}
	return nil
}
