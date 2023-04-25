package repositoryserver

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DefaultRepositoryPath                 = "kopia-repo-controller-test"
	DefaultRepositoryServerAdminUser      = "admin@test"
	DefaultRepositoryServerHost           = "localhost"
	DefaultRepositoryPassword             = "test1234"
	DefaultKanisterAdminUser              = "kanisterAdmin"
	DefaultKanisterUser                   = "kanisteruser"
	DefaultKanisterNamespace              = "kanister"
	DefaultKopiaRepositoryServerContainer = "repo-server-container"
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
				RootPath: DefaultRepositoryPath,
				Username: DefaultKanisterAdminUser,
				Hostname: DefaultRepositoryServerHost,
				PasswordSecretRef: v1.SecretReference{
					Namespace: namespace,
				},
			},
			Server: crv1alpha1.Server{
				UserAccess: crv1alpha1.UserAccess{
					UserAccessSecretRef: v1.SecretReference{
						Namespace: namespace,
					},
					Username: DefaultKanisterUser,
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
