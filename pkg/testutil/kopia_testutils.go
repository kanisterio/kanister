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
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	awsconfig "github.com/kanisterio/kanister/pkg/aws"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
)

const (
	DefaultRepositoryPath            = "/kopia-repo-controller-test/"
	DefaultRepositoryServerAdminUser = "admin@test"
	DefaultRepositoryServerHost      = "localhost"
	DefaultRepositoryPassword        = "test1234"
	DefaultKanisterAdminUser         = "kanisterAdmin"
	DefaultKanisterUser              = "kanisteruser"
)

func S3CredsLocationSecret() (*v1.Secret, *v1.Secret) {
	key := os.Getenv(awsconfig.AccessKeyID)
	val := os.Getenv(awsconfig.SecretAccessKey)
	s3Creds := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-s3-creds-",
		},
		Data: map[string][]byte{
			"aws_access_key_id":     []byte(key),
			"aws_secret_access_key": []byte(val),
		},
	}
	s3Location := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-s3-location-",
		},
		Data: map[string][]byte{
			"type":   []byte(crv1alpha1.LocationTypeS3Compliant),
			"bucket": []byte(TestS3BucketName),
			"path":   []byte(DefaultRepositoryPath),
			"region": []byte(TestS3Region),
		},
	}
	return s3Creds, s3Location
}

func KopiaRepositoryPassword() *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "repo-pass-",
		},
		Data: map[string][]byte{
			"repo-pass": []byte(DefaultRepositoryPassword),
		},
	}
}

func KopiaRepositoryServerAdminUser() *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "repository-admin-user-",
		},
		Data: map[string][]byte{
			"username": []byte(DefaultRepositoryServerAdminUser),
			"password": []byte(DefaultRepositoryPassword),
		},
	}
}

func KopiaRepositoryServerUserAccess() *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "repository-server-user-access-",
		},
		Data: map[string][]byte{
			DefaultRepositoryServerHost: []byte(DefaultRepositoryPassword),
		},
	}
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
				RootPath: DefaultRepositoryPath,
				Username: DefaultKanisterAdminUser,
				Hostname: DefaultRepositoryServerHost,
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
					Username: DefaultKanisterUser,
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
