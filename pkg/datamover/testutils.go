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
	"bytes"
	"context"
	crand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	awsconfig "github.com/kanisterio/kanister/pkg/aws"
	crclient "github.com/kanisterio/kanister/pkg/client/clientset/versioned/typed/cr/v1alpha1"
	kopiacmd "github.com/kanisterio/kanister/pkg/kopia/command"
	"github.com/kanisterio/kanister/pkg/kopia/repository"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
	"math/big"
	"os"
	"time"
)

const (
	repositoryServerTestNamespace  = "repository-server-test-namespace-"
	repositoryServerTestPod        = "repository-server-test-pod-"
	repositoryServerTestService    = "repository-server-test-service-"
	kanisterToolsImage             = "ghcr.io/kanisterio/kanister-tools:0.93.0"
	kanisterToolsImageEnvName      = "KANISTER_TOOLS"
	testAwsAccessKeyId             = "AKIAIOSFODNN7EXAMPLE"
	testAwsAccessSecretKey         = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	testAwsRegion                  = "us-west-2"
	testAwsLocationEndpoint        = "http://minio.minio.svc.cluster.local:9000"
	testAwsS3BucketName            = "tests.kanister.io"
	defaultKopiaRepositoryPassword = "test1234"
	defaultKopiaRepositoryUser     = "repositoryuser"
	defaultKopiaRepositoryPath     = "repository-server-test"
	defaultKopiaRepositoryHost     = "localhost"
)

func newTestClient() (*kubernetes.Clientset, error) {
	cfg, err := kube.LoadConfig()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to Load config")
	}

	testClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Clienset for k8s config")
	}

	_, err = crclient.NewForConfig(cfg)

	return testClient, nil
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

func getKanisterToolsImage() string {
	if val, ok := os.LookupEnv(kanisterToolsImageEnvName); ok {
		return val
	}
	return kanisterToolsImage
}

func createRepositoryServerTestPod(ctx context.Context, cli kubernetes.Interface, namespace string) (*corev1.Pod, error) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: repositoryServerTestPod,
			Namespace:    namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-repository-server-container",
					Image: getKanisterToolsImage(),
					Ports: []corev1.ContainerPort{
						{
							HostPort:      51515,
							ContainerPort: 51515,
						},
					},
					Env: []corev1.EnvVar{
						{
							Name:  awsconfig.AccessKeyID,
							Value: testAwsAccessKeyId,
						},
						{
							Name:  awsconfig.SecretAccessKey,
							Value: testAwsAccessSecretKey,
						},
						{
							Name:  awsconfig.Region,
							Value: testAwsRegion,
						},
						{
							Name:  "LOCATION_ENDPOINT",
							Value: testAwsLocationEndpoint,
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
			Selector: map[string]string{"name": name},
			Ports: []corev1.ServicePort{
				{
					Name:     "test-repo-server-service-port",
					Protocol: "TCP",
					Port:     51515,
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

func testS3CredsLocationSecret(ctx context.Context, cli kubernetes.Interface, namespace string) (*corev1.Secret, *corev1.Secret, error) {
	key := testAwsAccessKeyId
	val := testAwsAccessSecretKey
	s3Creds := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-s3-creds-",
		},
		Type: "secrets.kanister.io/aws",
		Data: map[string][]byte{
			"aws_access_key_id":     []byte(key),
			"aws_secret_access_key": []byte(val),
		},
	}
	s3Location := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-s3-location-",
		},
		Data: map[string][]byte{
			"type":     []byte(crv1alpha1.LocationTypeS3Compliant),
			"bucket":   []byte(testAwsS3BucketName),
			"path":     []byte(defaultKopiaRepositoryPath),
			"region":   []byte(testAwsRegion),
			"endpoint": []byte(testAwsLocationEndpoint),
		},
	}

	s3CredsCreated, err := cli.CoreV1().Secrets(namespace).Create(ctx, s3Creds, metav1.CreateOptions{})
	if err != nil {
		return nil, nil, err
	}
	s3LocationCreated, err := cli.CoreV1().Secrets(namespace).Create(ctx, s3Location, metav1.CreateOptions{})
	if err != nil {
		return nil, nil, err
	}

	return s3CredsCreated, s3LocationCreated, nil
}

func testKopiaTLSCertificate(ctx context.Context, cli kubernetes.Interface, namespace string) (*corev1.Secret, error) {
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization:  []string{"Test Organization"},
			Country:       []string{"Test Country"},
			Province:      []string{"Test Province"},
			Locality:      []string{"Test Locality"},
			StreetAddress: []string{"Test Street"},
			PostalCode:    []string{"123456"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 0, 1),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	caPrivKey, err := rsa.GenerateKey(crand.Reader, 4096)
	if err != nil {
		return nil, err
	}
	caBytes, err := x509.CreateCertificate(crand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, err
	}

	caPEM := new(bytes.Buffer)
	err = pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})
	if err != nil {
		return nil, err
	}

	caPrivKeyPEM := new(bytes.Buffer)
	err = pem.Encode(caPrivKeyPEM, &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey),
	})
	if err != nil {
		return nil, err
	}

	tls := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-repository-server-tls-cert-",
		},
		Type: "kubernetes.io/tls",
		Data: map[string][]byte{
			"tls.crt": caPEM.Bytes(),
			"tls.key": caPrivKeyPEM.Bytes(),
		},
	}

	tlsCreated, err := cli.CoreV1().Secrets(namespace).Create(ctx, tls, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	return tlsCreated, nil
}
