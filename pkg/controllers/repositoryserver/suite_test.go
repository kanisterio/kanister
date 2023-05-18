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
	crclientv1alpha1 "github.com/kanisterio/kanister/pkg/client/clientset/versioned/typed/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/poll"
	"github.com/kanisterio/kanister/pkg/resource"
	"github.com/pkg/errors"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type RepoServerControllerSuite struct {
	crCli                         crclientv1alpha1.CrV1alpha1Interface
	kubeCli                       kubernetes.Interface
	repoServerControllerNamespace string
	repoServerSecrets             repositoryServerSecrets
}

var _ = Suite(&RepoServerControllerSuite{})

func (s *RepoServerControllerSuite) SetUpSuite(c *C) {
	config, err := kube.LoadConfig()
	c.Assert(err, IsNil)
	cli, err := kubernetes.NewForConfig(config)
	c.Assert(err, IsNil)
	crCli, err := crclientv1alpha1.NewForConfig(config)
	c.Assert(err, IsNil)

	// Make sure the CRD's exist.
	_ = resource.CreateCustomResources(context.Background(), config)

	s.kubeCli = cli
	s.crCli = crCli

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme.Scheme,
		Port:               9443,
		MetricsBindAddress: "0",
	})
	c.Assert(err, IsNil)

	err = (&RepositoryServerReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr)
	c.Assert(err, IsNil)

	go func() {
		err = mgr.Start(ctrl.SetupSignalHandler())
		c.Assert(err, IsNil)
	}()

	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "repositoryservercontrollertest-",
		},
	}
	ctx := context.Background()
	cns, err := s.cli.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	s.repoServerControllerNamespace = cns.Name
	s.createRepositoryServerSecrets(c)
}

func (s *RepoServerControllerSuite) createRepositoryServerSecrets(c *C) {
	repoServerUserAccessSecretData := map[string][]byte{
		"localhost": []byte(DefaultKopiaRepositoryServerAccessPassword),
	}
	repoServerAdminSecretData := map[string][]byte{
		"username": []byte(DefaulKopiaRepositoryServerAdminUser),
		"password": []byte(DefaultKopiaRepositoryServerAdminPassword),
	}
	repoPasswordSecretData := map[string][]byte{
		repoPasswordKey: []byte(DefaultKopiaRepositoryPassword),
	}
	kopiaTLSSecretData, err := getKopiaTLSSecret()
	c.Assert(err, IsNil)
	s.repoServerSecrets = repositoryServerSecrets{}
	s.repoServerSecrets.serverUserAccess, err = s.createSecret("test-repository-server-user-access-", "", repoServerUserAccessSecretData)
	c.Assert(err, IsNil)
	s.repoServerSecrets.serverAdmin, err = s.createSecret("test-repository-server-admin-", "", repoServerAdminSecretData)
	c.Assert(err, IsNil)
	s.repoServerSecrets.repositoryPassword, err = s.createSecret("test-repository-password-", "", repoPasswordSecretData)
	c.Assert(err, IsNil)
	s.repoServerSecrets.serverTLS, err = s.createSecret("test-tls-", v1.SecretTypeTLS, kopiaTLSSecretData)
	c.Assert(err, IsNil)
	s.repoServerSecrets.storage, err = s.createSecret("test-repository-server-storage-", "", getDefaultS3StorageLocation())
	c.Assert(err, IsNil)
	s.repoServerSecrets.storageCredentials, err = s.createSecret("test-repository-server-storage-Creds-", "secrets.kanister.io/aws", getDefaultS3StorageCreds())
	c.Assert(err, IsNil)
}

func (s *RepoServerControllerSuite) createSecret(name string, secrettype v1.SecretType, data map[string][]byte) (se *v1.Secret, err error) {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: name,
		},
		Data: data,
	}
	if secrettype != "" {
		secret.Type = secrettype
	}

	se, err = s.cli.CoreV1().Secrets(s.repoServerControllerNamespace).Create(context.Background(), secret, metav1.CreateOptions{})
	return
}

func (s *RepoServerControllerSuite) TestRepositoryServerStatusIsServerReady(c *C) {
	repoServerCR := getDefaultKopiaRepositoryServerCR(s.repoServerControllerNamespace)
	setRepositoryServerSecretsInCR(&s.repoServerSecrets, repoServerCR)
	repoServerCRCreated, err := s.crCli.RepositoryServers(s.repoServerControllerNamespace).Create(context.Background(), repoServerCR, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	err = s.waitForRepoServerInfoUpdateInCR(repoServerCRCreated)
	c.Assert(err, IsNil)
	err = createKopiaRepository(s.cli, repoServerCRCreated, getDefaultS3StorageLocation())
	c.Assert(err, IsNil)
	err = s.waitOnRepositoryServerState(c, repoServerCRCreated)
	c.Assert(err, IsNil)
}

func (s *RepoServerControllerSuite) waitForRepoServerInfoUpdateInCR(rs *v1alpha1.RepositoryServer) error {
	ctxTimeout := 3 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
	defer cancel()
	err := poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		if rs.Status.ServerInfo.PodName == "" || rs.Status.ServerInfo.ServiceName == "" {
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
		if rs.Status.Progress == v1alpha1.ServerPending {
			return false, nil
		}
		if rs.Status.Progress == v1alpha1.ServerStopped {
			return false, errors.New(fmt.Sprintf(" There is failure in staring the repository server, server is in %s state, please check logs", rs.Status.Progress))
		}
		if rs.Status.Progress == v1alpha1.ServerReady {
			return true, nil
		}
		return false, errors.New(fmt.Sprintf("Unexpected Repository server state: %s", rs.Status.Progress))
	})

	return err
}

func (s *RepoServerControllerSuite) TearDownSuite(c *C) {
	if s.repoServerControllerNamespace != "" {
		err := s.cli.CoreV1().Namespaces().Delete(context.TODO(), s.repoServerControllerNamespace, metav1.DeleteOptions{})
		c.Assert(err, IsNil)
	}
}
