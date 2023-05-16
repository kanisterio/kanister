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

package testutil

import (
	"bytes"
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

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	awsconfig "github.com/kanisterio/kanister/pkg/aws"
)

const (
	// DefaultKopiaRepositoryPath is the default path for the kopia repository where the backups are stored.
	DefaultKopiaRepositoryPath = "kopia-repo-controller-test"
	// DefaultKopiaRepositoryServerAdminUser is the default admin user for the kopia repository server.
	DefaultKopiaRepositoryServerAdminUser = "admin@test"
	// DefaultKopiaRepositoryServerHost is the default host for the kopia repository server.
	DefaultKopiaRepositoryServerHost = "localhost"
	// DefaultKopiaRepositoryPassword is the default password for the kopia repository.
	DefaultKopiaRepositoryPassword = "test1234"
	// DefaultKopiaRepositoryUser is the default user for the kopia repository.
	DefaultKopiaRepositoryUser = "repositoryUser"
	// DefaultKopiaRepositoryServerAccessUser is the default user for the kopia repository server.
	DefaultKopiaRepositoryServerAccessUser = "kanisterUser"
	// DefaultKanisterNamespace is the default namespace for the kanister controller.
	DefaultKanisterNamespace = "kanister"
	// DefaultKopiaRepositoryServerAccessPassword is the default password for the kopia repository server.
	DefaultKopiaRepositoryServerAccessPassword = "test1234"
	// DefaultKopiaRepositoryServerAdminPassword is the default password for the kopia repository server admin.
	DefaultKopiaRepositoryServerAdminPassword = "admin1234"
	// DefaultKopiaRepositoryServerContainer is the default container for the kopia repository server.
	DefaultKopiaRepositoryServerContainer = "repo-server-container"
)

func S3CredsLocationSecret() (*v1.Secret, *v1.Secret) {
	key := os.Getenv(awsconfig.AccessKeyID)
	val := os.Getenv(awsconfig.SecretAccessKey)
	s3Creds := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-s3-creds-",
			Labels: map[string]string{
				"repo.kanister.io/target-namespace": "monitoring",
			},
		},
		Type: "secrets.kanister.io/aws",
		Data: map[string][]byte{
			"aws_access_key_id":     []byte(key),
			"aws_secret_access_key": []byte(val),
		},
	}
	s3Location := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-s3-location-",
			Labels: map[string]string{
				"repo.kanister.io/target-namespace": "monitoring",
			},
		},
		Data: map[string][]byte{
			"type":     []byte(crv1alpha1.LocationTypeS3Compliant),
			"bucket":   []byte(TestS3BucketName),
			"path":     []byte(DefaultKopiaRepositoryPath),
			"region":   []byte(TestS3Region),
			"endpoint": []byte(os.Getenv("LOCATION_ENDPOINT")),
		},
	}
	return s3Creds, s3Location
}

func KopiaRepositoryPassword() *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-repo-pass-",
		},
		Data: map[string][]byte{
			"repo-password": []byte(DefaultKopiaRepositoryPassword),
		},
	}
}

func KopiaRepositoryServerAdminUser() *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-repository-admin-user-",
		},
		Data: map[string][]byte{
			"username": []byte(DefaultKopiaRepositoryServerAdminUser),
			"password": []byte(DefaultKopiaRepositoryServerAdminPassword),
		},
	}
}

func KopiaRepositoryServerUserAccess() *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-repository-server-user-access-",
		},
		Data: map[string][]byte{
			DefaultKopiaRepositoryServerHost: []byte(DefaultKopiaRepositoryServerAccessPassword),
		},
	}
}

func KopiaTLSCertificate() (*v1.Secret, error) {
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

	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-repository-server-tls-cert-",
		},
		Type: "kubernetes.io/tls",
		Data: map[string][]byte{
			"tls.crt": caPEM.Bytes(),
			"tls.key": caPrivKeyPEM.Bytes(),
		},
	}, nil
}

func NewKopiaRepositoryServer() *crv1alpha1.RepositoryServer {
	return &crv1alpha1.RepositoryServer{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-kopia-repo-server-",
		},
		Spec: crv1alpha1.RepositoryServerSpec{
			Storage: crv1alpha1.Storage{
				SecretRef: v1.SecretReference{
					Name:      "",
					Namespace: "",
				},
				CredentialSecretRef: v1.SecretReference{
					Name:      "",
					Namespace: "",
				},
			},
			Repository: crv1alpha1.Repository{
				RootPath: DefaultKopiaRepositoryPath,
				Username: DefaultKopiaRepositoryUser,
				Hostname: DefaultKopiaRepositoryServerHost,
				PasswordSecretRef: v1.SecretReference{
					Name:      "",
					Namespace: "",
				},
			},
			Server: crv1alpha1.Server{
				UserAccess: crv1alpha1.UserAccess{
					UserAccessSecretRef: v1.SecretReference{
						Name:      "",
						Namespace: "",
					},
					Username: DefaultKopiaRepositoryServerAccessUser,
				},
				AdminSecretRef: v1.SecretReference{
					Name:      "",
					Namespace: "",
				},
				TLSSecretRef: v1.SecretReference{
					Name:      "",
					Namespace: "",
				},
			},
		},
	}
}
