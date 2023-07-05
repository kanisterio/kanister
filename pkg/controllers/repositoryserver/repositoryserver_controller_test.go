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
	"context"
	"fmt"
	"testing"
	"time"

	. "gopkg.in/check.v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	crclientv1alpha1 "github.com/kanisterio/kanister/pkg/client/clientset/versioned/typed/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/poll"
	"github.com/kanisterio/kanister/pkg/resource"
	"github.com/kanisterio/kanister/pkg/secrets"
	"github.com/kanisterio/kanister/pkg/secrets/repositoryserver"
	reposerver "github.com/kanisterio/kanister/pkg/secrets/repositoryserver"
	"github.com/kanisterio/kanister/pkg/testutil"
	"github.com/pkg/errors"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type RepoServerControllerSuite struct {
	crCli                         crclientv1alpha1.CrV1alpha1Interface
	kubeCli                       kubernetes.Interface
	repoServerControllerNamespace string
	repoServerSecrets             repositoryServerSecrets
	DefaultRepoServerReconciler   *RepositoryServerReconciler
}

var _ = Suite(&RepoServerControllerSuite{})

func (s *RepoServerControllerSuite) SetUpSuite(c *C) {
	config, err := kube.LoadConfig()
	c.Assert(err, IsNil)
	cli, err := kubernetes.NewForConfig(config)
	c.Assert(err, IsNil)
	crCli, err := crclientv1alpha1.NewForConfig(config)
	c.Assert(err, IsNil)

	// Make sure the CRDs exist.
	err = resource.CreateCustomResources(context.Background(), config)
	c.Assert(err, IsNil)
	err = resource.CreateRepoServerCustomResource(context.Background(), config)
	c.Assert(err, IsNil)

	s.kubeCli = cli
	s.crCli = crCli

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme.Scheme,
		Port:               9443,
		MetricsBindAddress: "0",
	})
	c.Assert(err, IsNil)

	repoReconciler := &RepositoryServerReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}

	err = repoReconciler.SetupWithManager(mgr)
	c.Assert(err, IsNil)

	go func() {
		err = mgr.Start(ctrl.SetupSignalHandler())
		c.Assert(err, IsNil)
	}()
	s.DefaultRepoServerReconciler = repoReconciler
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "repositoryservercontrollertest-",
		},
	}
	cns, err := s.kubeCli.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	s.repoServerControllerNamespace = cns.Name
	s.createRepositoryServerSecrets(c)
}

func (s *RepoServerControllerSuite) createRepositoryServerSecrets(c *C) {
	kopiaTLSSecretData, err := testutil.GetKopiaTLSSecretData()
	c.Assert(err, IsNil)

	s.repoServerSecrets = repositoryServerSecrets{}
	s.repoServerSecrets.serverUserAccess, err = s.CreateRepositoryServerUserAccessSecret(testutil.GetRepoServerUserAccessSecretData("localhost", testutil.KopiaRepositoryServerAccessPassword))
	c.Assert(err, IsNil)

	s.repoServerSecrets.serverAdmin, err = s.CreateRepositoryServerAdminSecret(testutil.GetRepoServerAdminSecretData(testutil.KopiaRepositoryServerAdminUser, testutil.KopiaRepositoryServerAdminPassword))
	c.Assert(err, IsNil)

	s.repoServerSecrets.repositoryPassword, err = s.CreateRepositoryPasswordSecret(testutil.GetRepoPasswordSecretData(testutil.KopiaRepositoryPassword))
	c.Assert(err, IsNil)

	s.repoServerSecrets.serverTLS, err = s.CreateKopiaTLSSecret(kopiaTLSSecretData)
	c.Assert(err, IsNil)

	s.repoServerSecrets.storage, err = s.CreateStorageLocationSecret(testutil.GetDefaultS3CompliantStorageLocation())
	c.Assert(err, IsNil)

	s.repoServerSecrets.storageCredentials, err = s.CreateAWSStorageCredentialsSecret(testutil.GetDefaultS3StorageCreds())
	c.Assert(err, IsNil)
}

func (s *RepoServerControllerSuite) CreateRepositoryServerAdminSecret(data map[string][]byte) (se *v1.Secret, err error) {
	return testutil.CreateSecret(s.kubeCli, s.repoServerControllerNamespace, "test-repository-server-admin-", reposerver.AdminCredentialsSecret, data)
}

func (s *RepoServerControllerSuite) CreateRepositoryServerUserAccessSecret(data map[string][]byte) (se *v1.Secret, err error) {
	return testutil.CreateSecret(s.kubeCli, s.repoServerControllerNamespace, "test-repository-server-user-access-", "", data)
}

func (s *RepoServerControllerSuite) CreateRepositoryPasswordSecret(data map[string][]byte) (se *v1.Secret, err error) {
	return testutil.CreateSecret(s.kubeCli, s.repoServerControllerNamespace, "test-repository-password-", reposerver.RepositoryPasswordSecret, data)
}

func (s *RepoServerControllerSuite) CreateKopiaTLSSecret(data map[string][]byte) (se *v1.Secret, err error) {
	return testutil.CreateSecret(s.kubeCli, s.repoServerControllerNamespace, "test-kopia-tls-", v1.SecretTypeTLS, data)
}

func (s *RepoServerControllerSuite) CreateStorageLocationSecret(data map[string][]byte) (se *v1.Secret, err error) {
	return testutil.CreateSecret(s.kubeCli, "test-repository-server-storage-", s.repoServerControllerNamespace, reposerver.Location, data)
}

func (s *RepoServerControllerSuite) CreateAWSStorageCredentialsSecret(data map[string][]byte) (se *v1.Secret, err error) {
	return testutil.CreateSecret(s.kubeCli, "test-repository-server-storage-creds-", s.repoServerControllerNamespace, v1.SecretType(secrets.AWSSecretType), data)
}

func (s *RepoServerControllerSuite) CreateAzureStorageCredentialsSecret(data map[string][]byte) (se *v1.Secret, err error) {
	return testutil.CreateSecret(s.kubeCli, "test-repository-server-storage-creds-", s.repoServerControllerNamespace, v1.SecretType(secrets.AzureSecretType), data)
}

func (s *RepoServerControllerSuite) CreateGCPStorageCredentialsSecret(data map[string][]byte) (se *v1.Secret, err error) {
	return testutil.CreateSecret(s.kubeCli, "test-repository-server-storage-creds-", s.repoServerControllerNamespace, v1.SecretType(secrets.GCPSecretType), data)
}

func (s *RepoServerControllerSuite) TestRepositoryServerImmutability(c *C) {
	// Create a repository server CR.
	repoServerCR := testutil.GetTestKopiaRepositoryServerCR(s.repoServerControllerNamespace)
	setRepositoryServerSecretsInCR(&s.repoServerSecrets, repoServerCR)
	repoServerCRCreated, err := s.crCli.RepositoryServers(s.repoServerControllerNamespace).Create(context.Background(), repoServerCR, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	// Update the repository server CR's Immutable field.
	repoServerCRCreated.Spec.Repository.RootPath = "/updated-test-path/"
	_, err = s.crCli.RepositoryServers(s.repoServerControllerNamespace).Update(context.Background(), repoServerCRCreated, metav1.UpdateOptions{})
	// Expect an error.
	c.Assert(err, NotNil)
	// Delete the repository server CR.
	err = s.crCli.RepositoryServers(s.repoServerControllerNamespace).Delete(context.Background(), repoServerCRCreated.Name, metav1.DeleteOptions{})
	c.Assert(err, IsNil)
}

func (s *RepoServerControllerSuite) TestRepositoryServerStatusIsServerReady(c *C) {
	// Test if the repository server CR state is Ready
	// with all the right configuration
	repoServerCR := testutil.GetTestKopiaRepositoryServerCR(s.repoServerControllerNamespace)
	setRepositoryServerSecretsInCR(&s.repoServerSecrets, repoServerCR)
	repoServerCRCreated, err := s.crCli.RepositoryServers(s.repoServerControllerNamespace).Create(context.Background(), repoServerCR, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	err = s.waitForRepoServerInfoUpdateInCR(repoServerCRCreated.Name)
	c.Assert(err, IsNil)
	err = testutil.CreateTestKopiaRepository(s.kubeCli, repoServerCRCreated, testutil.GetDefaultS3CompliantStorageLocation())
	c.Assert(err, IsNil)
	err = s.waitOnRepositoryServerState(c, repoServerCRCreated)
	c.Assert(err, IsNil)
}

func (s *RepoServerControllerSuite) TestRepositoryServerCRStateWithoutSecrets(c *C) {
	// Test if server is stopped
	// when no storage secrets are set
	repoServerCR := testutil.GetTestKopiaRepositoryServerCR(s.repoServerControllerNamespace)
	repoServerCRCreated, err := s.crCli.RepositoryServers(s.repoServerControllerNamespace).Create(context.Background(), repoServerCR, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	err = s.waitForRepoServerInfoUpdateInCR(repoServerCRCreated.Name)
	c.Assert(err, IsNil)
	err = testutil.CreateTestKopiaRepository(s.kubeCli, repoServerCRCreated, testutil.GetDefaultS3CompliantStorageLocation())
	c.Assert(err, IsNil)
	err = s.waitOnRepositoryServerState(c, repoServerCRCreated)
	c.Assert(err, NotNil)
	c.Assert(repoServerCRCreated.Status.Progress, Equals, v1alpha1.Failed)
}

func (s *RepoServerControllerSuite) TestCreationOfOwnedResources(c *C) {
	// Test if pod and service for repository server
	// is created successfully
	ctx := context.Background()
	repoServerCR := testutil.GetTestKopiaRepositoryServerCR(s.repoServerControllerNamespace)
	setRepositoryServerSecretsInCR(&s.repoServerSecrets, repoServerCR)
	repoServerCRCreated, err := s.crCli.RepositoryServers(s.repoServerControllerNamespace).Create(ctx, repoServerCR, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	err = s.waitForRepoServerInfoUpdateInCR(repoServerCRCreated.Name)
	c.Assert(err, IsNil)
	pod, err := s.kubeCli.CoreV1().Pods(s.repoServerControllerNamespace).Get(ctx, repoServerCRCreated.Status.ServerInfo.PodName, metav1.GetOptions{})
	c.Assert(err, IsNil)
	c.Assert(len(pod.OwnerReferences), Equals, 1)
	c.Assert(pod.OwnerReferences[0].UID, Equals, repoServerCRCreated.UID)
	service, err := s.kubeCli.CoreV1().Services(s.repoServerControllerNamespace).Get(ctx, repoServerCRCreated.Status.ServerInfo.ServiceName, metav1.GetOptions{})
	c.Assert(err, IsNil)
	c.Assert(len(service.OwnerReferences), Equals, 1)
	c.Assert(service.OwnerReferences[0].UID, Equals, repoServerCRCreated.UID)
}

func (s *RepoServerControllerSuite) TestInvalidRepositoryPassword(c *C) {
	ctx := context.Background()
	originalrepoServerCR := testutil.GetTestKopiaRepositoryServerCR(s.repoServerControllerNamespace)
	setRepositoryServerSecretsInCR(&s.repoServerSecrets, originalrepoServerCR)
	for _, tc := range []struct {
		description  string
		testFunction func(rs *v1alpha1.RepositoryServer)
	}{
		{
			description: "Invalid Repository Password",
			testFunction: func(rs *v1alpha1.RepositoryServer) {
				InvalidRepositoryPassword, err := s.CreateRepositoryPasswordSecret(testutil.GetRepoPasswordSecretData("invalidPassword"))
				c.Assert(err, IsNil)
				rs.Spec.Repository.PasswordSecretRef.Name = InvalidRepositoryPassword.Name
				rs.Spec.Repository.PasswordSecretRef.Namespace = InvalidRepositoryPassword.Namespace
			},
		},
		{
			description: "Invalid Storage Location",
			testFunction: func(rs *v1alpha1.RepositoryServer) {
				storageLocationData := testutil.GetDefaultS3CompliantStorageLocation()
				storageLocationData[repositoryserver.BucketKey] = []byte("invalidbucket")
				InvalidStorageLocationSecret, err := s.CreateStorageLocationSecret(storageLocationData)
				c.Assert(err, IsNil)
				rs.Spec.Storage.SecretRef.Name = InvalidStorageLocationSecret.Name
				rs.Spec.Storage.SecretRef.Namespace = InvalidStorageLocationSecret.Namespace
			},
		},
		{
			description: "Invalid Storage location credentials",
			testFunction: func(rs *v1alpha1.RepositoryServer) {
				storageLocationCredsData := testutil.GetDefaultS3StorageCreds()
				storageLocationCredsData[secrets.AWSAccessKeyID] = []byte("testaccesskey")
				InvalidStorageLocationCrdesSecret, err := s.CreateStorageLocationSecret(storageLocationCredsData)
				c.Assert(err, IsNil)
				rs.Spec.Storage.CredentialSecretRef.Name = InvalidStorageLocationCrdesSecret.Name
				rs.Spec.Storage.CredentialSecretRef.Namespace = InvalidStorageLocationCrdesSecret.Namespace

			},
		},
	} {

		invalidCR := *originalrepoServerCR
		tc.testFunction(&invalidCR)
		repoServerCRCreated, err := s.crCli.RepositoryServers(s.repoServerControllerNamespace).Create(ctx, &invalidCR, metav1.CreateOptions{})
		c.Assert(err, IsNil)
		err = s.waitForRepoServerInfoUpdateInCR(repoServerCRCreated.Name)
		c.Assert(err, IsNil)
		c.Assert(repoServerCRCreated.Status.Progress, Equals, v1alpha1.Failed)
	}

}

func (s *RepoServerControllerSuite) waitForRepoServerInfoUpdateInCR(repoServerName string) error {
	ctxTimeout := 3 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
	defer cancel()
	err := poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		repoServerCR, err := s.crCli.RepositoryServers(s.repoServerControllerNamespace).Get(ctx, repoServerName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if repoServerCR.Status.ServerInfo.PodName == "" || repoServerCR.Status.ServerInfo.ServiceName == "" {
			return false, errors.New("Repository server CR server not set")
		}
		return true, nil
	})
	return err
}

func (s *RepoServerControllerSuite) waitOnRepositoryServerState(c *C, rs *v1alpha1.RepositoryServer) error {
	ctxTimeout := 10 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
	defer cancel()
	err := poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		if rs.Status.Progress == v1alpha1.Pending {
			return false, nil
		}
		if rs.Status.Progress == v1alpha1.Failed {
			return false, errors.New(fmt.Sprintf(" There is failure in staring the repository server, server is in %s state, please check logs", rs.Status.Progress))
		}
		if rs.Status.Progress == v1alpha1.Ready {
			return true, nil
		}
		return false, errors.New(fmt.Sprintf("Unexpected Repository server state: %s", rs.Status.Progress))
	})

	return err
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

func (s *RepoServerControllerSuite) TearDownSuite(c *C) {
	if s.repoServerControllerNamespace != "" {
		err := s.kubeCli.CoreV1().Namespaces().Delete(context.TODO(), s.repoServerControllerNamespace, metav1.DeleteOptions{})
		c.Assert(err, IsNil)
	}
}
