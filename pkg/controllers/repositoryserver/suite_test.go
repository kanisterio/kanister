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
	"testing"

	. "gopkg.in/check.v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	crclientv1alpha1 "github.com/kanisterio/kanister/pkg/client/clientset/versioned/typed/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/resource"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type RepoServerControllerSuite struct {
	testEnv                       *envtest.Environment
	crCli                         crclientv1alpha1.CrV1alpha1Interface
	cli                           kubernetes.Interface
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

	s.cli = cli
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
		"localhost": []byte("test1234"),
	}
	repoServerAdminSecretData := map[string][]byte{
		"username": []byte("admin@testpod1"),
		"password": []byte("test1234"),
	}
	repoPasswordSecretData := map[string][]byte{
		"repo-password": []byte("test1234"),
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

func (s *RepoServerControllerSuite) TestCreationOfOwnedResources(c *C) {
	repoServerCR := getDefaultKopiaRepositoryServerCR(s.repoServerControllerNamespace)
	setRepositoryServerSecretsInCR(&s.repoServerSecrets, repoServerCR)
}

func (s *RepoServerControllerSuite) TearDownSuite(c *C) {
	if s.repoServerControllerNamespace != "" {
		err := s.cli.CoreV1().Namespaces().Delete(context.TODO(), s.repoServerControllerNamespace, metav1.DeleteOptions{})
		c.Assert(err, IsNil)
	}
}
