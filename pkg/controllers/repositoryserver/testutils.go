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

	"github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	awsconfig "github.com/kanisterio/kanister/pkg/aws"
	"github.com/kanisterio/kanister/pkg/kopia/command"
	"github.com/kanisterio/kanister/pkg/kopia/repository"
	"github.com/kanisterio/kanister/pkg/secrets"
	"github.com/kanisterio/kanister/pkg/secrets/repositoryserver"
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
		repositoryserver.LocationTypeKey: []byte(crv1alpha1.LocationTypeS3Compliant),
		repositoryserver.BucketKey:       []byte(testutil.TestS3BucketName),
		pathKey:                          []byte(defaultKopiaRepositoryPath),
		repositoryserver.RegionKey:       []byte(testutil.TestS3Region),
		repositoryserver.EndpointKey:     []byte(os.Getenv("LOCATION_ENDPOINT")),
	}
}

func getRepoPasswordSecretData(password string) map[string][]byte {
	return map[string][]byte{
		repoPasswordKey: []byte(password),
	}
}

func getRepoServerAdminSecretData(username, password string) map[string][]byte {
	return map[string][]byte{
		serverAdminUserNameKey: []byte(username),
		serverAdminPasswordKey: []byte(password),
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
	return createSecret(cli, namespace, "test-repository-server-admin-", repositoryserver.RepositoryServerAdminCredentials, data)
}

func createRepositoryServerUserAccessSecret(cli kubernetes.Interface, namespace string, data map[string][]byte) (se *v1.Secret, err error) {
	return createSecret(cli, "test-repository-server-user-access-", namespace, "", data)
}

func createRepositoryPassword(cli kubernetes.Interface, namespace string, data map[string][]byte) (se *v1.Secret, err error) {
	return createSecret(cli, "test-repository-password-", namespace, repositoryserver.RepositoryPassword, data)
}

func createKopiaTLSSecret(cli kubernetes.Interface, namespace string, data map[string][]byte) (se *v1.Secret, err error) {
	return createSecret(cli, "test-repository-password-", namespace, v1.SecretTypeTLS, data)
}

func createStorageLocationSecret(cli kubernetes.Interface, namespace string, data map[string][]byte) (se *v1.Secret, err error) {
	return createSecret(cli, "test-repository-server-storage-", namespace, repositoryserver.Location, data)
}

func createKopiaRepository(cli kubernetes.Interface, rs *v1alpha1.RepositoryServer, storageLocation map[string][]byte) error {
	contentCacheMB, metadataCacheMB := command.GetGeneralCacheSizeSettings()

	commandArgs := command.RepositoryCommandArgs{
		CommandArgs: &command.CommandArgs{
			RepoPassword:   defaultKopiaRepositoryPassword,
			ConfigFilePath: command.DefaultConfigFilePath,
			LogDirectory:   command.DefaultLogDirectory,
		},
		CacheDirectory:  command.DefaultCacheDirectory,
		Hostname:        defaultKopiaRepositoryServerHost,
		ContentCacheMB:  contentCacheMB,
		MetadataCacheMB: metadataCacheMB,
		Username:        defaultKopiaRepositoryUser,
		RepoPathPrefix:  defaultKopiaRepositoryPath,
		Location:        storageLocation,
	}
	return repository.CreateKopiaRepository(
		cli,
		defaultKanisterNamespace,
		rs.Status.ServerInfo.PodName,
		defaultKopiaRepositoryServerContainer,
		commandArgs,
	)
}

func getDefaultKopiaRepositoryServerCR(namespace string) *crv1alpha1.RepositoryServer {
	repositoryServer := &crv1alpha1.RepositoryServer{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-kopia-repo-server-",
			Namespace:    namespace,
		},
		Spec: crv1alpha1.RepositoryServerSpec{
			Storage: crv1alpha1.Storage{
				SecretRef: v1.SecretReference{
					Namespace: namespace,
				},
				CredentialSecretRef: v1.SecretReference{
					Namespace: namespace,
				},
			},
			Repository: crv1alpha1.Repository{
				RootPath: defaultKopiaRepositoryPath,
				Username: defaultKopiaRepositoryUser,
				Hostname: defaultKopiaRepositoryServerHost,
				PasswordSecretRef: v1.SecretReference{
					Namespace: namespace,
				},
			},
			Server: crv1alpha1.Server{
				UserAccess: crv1alpha1.UserAccess{
					UserAccessSecretRef: v1.SecretReference{
						Namespace: namespace,
					},
					Username: defaultKopiaRepositoryServerAccessUser,
				},
				AdminSecretRef: v1.SecretReference{
					Namespace: namespace,
				},
				TLSSecretRef: v1.SecretReference{
					Namespace: namespace,
				},
			},
		},
	}
	return repositoryServer
}

func setRepositoryServerSecretsInCR(secrets *repositoryServerSecrets, repoServerCR *crv1alpha1.RepositoryServer) {
	if secrets != nil {
		if secrets.serverAdmin != nil {
			repoServerCR.Spec.Server.AdminSecretRef.Name = secrets.serverAdmin.Name
		}
		if secrets.repositoryPassword != nil {
			repoServerCR.Spec.Repository.PasswordSecretRef.Name = secrets.repositoryPassword.Name
		}

		if secrets.serverUserAccess != nil {
			repoServerCR.Spec.Server.UserAccess.UserAccessSecretRef.Name = secrets.serverUserAccess.Name
		}
		if secrets.serverTLS != nil {
			repoServerCR.Spec.Server.TLSSecretRef.Name = secrets.serverTLS.Name
		}
		if secrets.storage != nil {
			repoServerCR.Spec.Storage.SecretRef.Name = secrets.storage.Name
		}
		if secrets.storageCredentials != nil {
			repoServerCR.Spec.Storage.CredentialSecretRef.Name = secrets.storageCredentials.Name
		}
	}
}
