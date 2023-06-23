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
	"os"

	"github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	awsconfig "github.com/kanisterio/kanister/pkg/aws"
	"github.com/kanisterio/kanister/pkg/kopia/command"
	"github.com/kanisterio/kanister/pkg/kopia/repository"
	"github.com/kanisterio/kanister/pkg/secrets"
	reposerver "github.com/kanisterio/kanister/pkg/secrets/repositoryserver"
	"github.com/kanisterio/kanister/pkg/testutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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
