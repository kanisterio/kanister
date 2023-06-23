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
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	awsconfig "github.com/kanisterio/kanister/pkg/aws"
	"github.com/kanisterio/kanister/pkg/secrets"
	reposerver "github.com/kanisterio/kanister/pkg/secrets/repositoryserver"
	"github.com/kanisterio/kanister/pkg/testutil"
)

const (
	defaultKopiaRepositoryPath                 = "kopia-repo-controller-test"
	defaulKopiaRepositoryServerAdminUser       = "admin@test"
	defaultKopiaRepositoryServerAdminPassword  = "admin1234"
	defaultKopiaRepositoryServerHost           = "localhost"
	defaultKopiaRepositoryPassword             = "test1234"
	defaultKopiaRepositoryUser                 = "repository-user"
	defaultKopiaRepositoryServerAccessUser     = "kanister-user"
	defaultKopiaRepositoryServerAccessPassword = "test1234"
	defaultKanisterNamespace                   = "kanister"
	defaultKopiaRepositoryServerContainer      = "repo-server-container"
	pathKey                                    = "path"
)

func getKopiaTLSSecret() (map[string][]byte, error) {
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
	caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
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

	return map[string][]byte{
		"tls.crt": caPEM.Bytes(),
		"tls.key": caPrivKeyPEM.Bytes(),
	}, nil
}

func getDefaultS3StorageCreds() map[string][]byte {
	key := os.Getenv(awsconfig.AccessKeyID)
	val := os.Getenv(awsconfig.SecretAccessKey)

	return map[string][]byte{
		secrets.AWSAccessKeyID:     []byte(key),
		secrets.AWSSecretAccessKey: []byte(val),
	}
}

func getDefaultS3CompliantStorageLocation() map[string][]byte {
	return map[string][]byte{
		reposerver.TypeKey:     []byte(crv1alpha1.LocationTypeS3Compliant),
		reposerver.BucketKey:   []byte(testutil.TestS3BucketName),
		pathKey:                []byte(defaultKopiaRepositoryPath),
		reposerver.RegionKey:   []byte(testutil.TestS3Region),
		reposerver.EndpointKey: []byte(os.Getenv("LOCATION_ENDPOINT")),
	}
}

func getRepoPasswordSecretData(password string) map[string][]byte {
	return map[string][]byte{
		reposerver.RepoPasswordKey: []byte(password),
	}
}

func getRepoServerAdminSecretData(username, password string) map[string][]byte {
	return map[string][]byte{
		reposerver.AdminUsernameKey: []byte(username),
		reposerver.AdminPasswordKey: []byte(password),
	}
}

func getRepoServerUserAccessSecretData(hostname, password string) map[string][]byte {
	return map[string][]byte{
		hostname: []byte(password),
	}
}

func createSecret(cli kubernetes.Interface, namespace, name string, secrettype v1.SecretType, data map[string][]byte) (se *v1.Secret, err error) {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: name,
		},
		Data: data,
	}
	if secrettype != "" {
		secret.Type = secrettype
	}

	se, err = cli.CoreV1().Secrets(namespace).Create(context.Background(), secret, metav1.CreateOptions{})
	return
}

func createRepositoryServerAdminSecret(cli kubernetes.Interface, namespace string, data map[string][]byte) (se *v1.Secret, err error) {
	return createSecret(cli, namespace, "test-repository-server-admin-", reposerver.AdminCredentialsSecret, data)
}

func createRepositoryServerUserAccessSecret(cli kubernetes.Interface, namespace string, data map[string][]byte) (se *v1.Secret, err error) {
	return createSecret(cli, "test-repository-server-user-access-", namespace, "", data)
}

func createRepositoryPassword(cli kubernetes.Interface, namespace string, data map[string][]byte) (se *v1.Secret, err error) {
	return createSecret(cli, "test-repository-password-", namespace, reposerver.RepositoryPasswordSecret, data)
}

func createKopiaTLSSecret(cli kubernetes.Interface, namespace string, data map[string][]byte) (se *v1.Secret, err error) {
	return createSecret(cli, "test-repository-password-", namespace, v1.SecretTypeTLS, data)
}

func createStorageLocationSecret(cli kubernetes.Interface, namespace string, data map[string][]byte) (se *v1.Secret, err error) {
	return createSecret(cli, "test-repository-server-storage-", namespace, reposerver.Location, data)
}
