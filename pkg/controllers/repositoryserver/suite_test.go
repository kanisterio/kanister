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

	crclientv1alpha1 "github.com/kanisterio/kanister/pkg/client/clientset/versioned/typed/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/resource"
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
	cns, err := s.kubeCli.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	s.repoServerControllerNamespace = cns.Name
	s.createRepositoryServerSecrets(c)
}

func (s *RepoServerControllerSuite) createRepositoryServerSecrets(c *C) {
	kopiaTLSSecretData, err := getKopiaTLSSecret()
	c.Assert(err, IsNil)
	s.repoServerSecrets = repositoryServerSecrets{}
	s.repoServerSecrets.serverUserAccess, err = createRepositoryServerUserAccessSecret(s.kubeCli, s.repoServerControllerNamespace, getRepoServerUserAccessSecretData("localhost", defaultKopiaRepositoryServerAccessPassword))
	c.Assert(err, IsNil)
	s.repoServerSecrets.serverAdmin, err = createRepositoryServerAdminSecret(s.kubeCli, s.repoServerControllerNamespace, getRepoServerAdminSecretData(defaulKopiaRepositoryServerAdminUser, defaultKopiaRepositoryServerAdminPassword))
	c.Assert(err, IsNil)
	s.repoServerSecrets.repositoryPassword, err = createRepositoryPassword(s.kubeCli, s.repoServerControllerNamespace, getRepoPasswordSecretData(defaultKopiaRepositoryPassword))
	c.Assert(err, IsNil)
	s.repoServerSecrets.serverTLS, err = createKopiaTLSSecret(s.kubeCli, s.repoServerControllerNamespace, kopiaTLSSecretData)
	c.Assert(err, IsNil)
	s.repoServerSecrets.storage, err = createStorageLocationSecret(s.kubeCli, s.repoServerControllerNamespace, getDefaultS3CompliantStorageLocation())
	c.Assert(err, IsNil)
	s.repoServerSecrets.storageCredentials, err = createSecret(s.kubeCli, "test-repository-server-storage-creds-", s.repoServerControllerNamespace, "secrets.kanister.io/aws", getDefaultS3StorageCreds())
	c.Assert(err, IsNil)
}

func (s *RepoServerControllerSuite) TearDownSuite(c *C) {
	if s.repoServerControllerNamespace != "" {
		err := s.kubeCli.CoreV1().Namespaces().Delete(context.TODO(), s.repoServerControllerNamespace, metav1.DeleteOptions{})
		c.Assert(err, IsNil)
	}
}
