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
	"path/filepath"
	"testing"

	. "gopkg.in/check.v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	crkanisteriov1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type ControllerSuite struct {
	testEnv *envtest.Environment
}

var _ = Suite(&ControllerSuite{})

func (s *ControllerSuite) SetUpSuite(c *C) {
	c.Log("Bootstrapping test environment with Kanister CRDs")
	useExistingCluster := true
	s.testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "customresource")},
		ErrorIfCRDPathMissing: true,
		UseExistingCluster:    &useExistingCluster,
	}

	cfg, err := s.testEnv.Start()
	c.Assert(err, IsNil)
	c.Assert(cfg, NotNil)

	err = crkanisteriov1alpha1.AddToScheme(scheme.Scheme)
	c.Assert(err, IsNil)

	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	c.Assert(err, IsNil)
	c.Assert(k8sClient, NotNil)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme.Scheme,
	})
	c.Assert(err, IsNil)

	err = (&RepositoryServerReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr)
	c.Assert(err, IsNil)

	err = mgr.Start(ctrl.SetupSignalHandler())
	c.Assert(err, IsNil)
}

func (s *ControllerSuite) SetupTest(c *C) {

}

func (s *ControllerSuite) TestCreationOfOwnedResources(c *C) {

}

func (s *ControllerSuite) TearDownSuite(c *C) {
	if s.testEnv != nil {
		c.Log("Tearing down the test environment")
		err := s.testEnv.Stop()
		c.Assert(err, IsNil)
	}
}
