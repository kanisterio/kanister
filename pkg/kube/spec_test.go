// +build !unit

package kube_test

import (
	"context"

	. "gopkg.in/check.v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/kube"
	kubetest "github.com/kanisterio/kanister/pkg/kube/test"
)

type SpecSuite struct {
	cli       kubernetes.Interface
	namespace string
}

var _ = Suite(&SpecSuite{})

func (s *SpecSuite) SetUpSuite(c *C) {
	c.Skip("Too slow")
	s.cli = kube.NewClient()
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "spectest-",
		},
	}
	cns, err := s.cli.Core().Namespaces().Create(ns)
	c.Assert(err, IsNil)
	s.namespace = cns.Name
}

func (s *SpecSuite) TearDownSuite(c *C) {
	if s.namespace != "" {
		s.cli.Core().Namespaces().Delete(s.namespace, nil)
	}
}

func (s *SpecSuite) TestRecreateStatefulSet(c *C) {
	// Create statefulset
	ctx := context.Background()
	ss := kubetest.CreateStatefulSetFromYamlSpec(ctx, c, s.cli, s.namespace, specFile)

	// Check the stateful set is returned
	ssets, err := s.cli.AppsV1beta1().StatefulSets(s.namespace).List(metav1.ListOptions{})
	c.Assert(ssets.Items, HasLen, 1)

	// Create a spec for the statefulset
	ssSpec, err := kube.CreateSpec(ss)
	c.Assert(err, IsNil)

	// Delete the statefulset
	now := int64(0)
	err = s.cli.AppsV1beta1().StatefulSets(s.namespace).Delete(ss.Name, &metav1.DeleteOptions{GracePeriodSeconds: &now})
	c.Assert(err, IsNil)

	// Check the stateful set is not returned
	ssets, err = s.cli.AppsV1beta1().StatefulSets(s.namespace).List(metav1.ListOptions{})
	c.Assert(ssets.Items, HasLen, 0)

	// Re-create the statefulset from the spec
	newSS, err := kube.GetStatefulSetFromSpec(ssSpec)
	c.Assert(err, IsNil)
	newSS, err = s.cli.AppsV1beta1().StatefulSets(s.namespace).Create(newSS)
	c.Assert(err, IsNil)
	c.Assert(newSS.Name, Equals, ss.Name)

	// Check the stateful set is returned
	ssets, err = s.cli.AppsV1beta1().StatefulSets(s.namespace).List(metav1.ListOptions{})
	c.Assert(ssets.Items, HasLen, 1)
}

func (s *SpecSuite) TestRecreateDeployment(c *C) {
	// Create the deployment
	dep := kubetest.CreateDeployment(c, s.cli, s.namespace, map[string]string{"d": "1"}, map[string]string{}, 1)

	// Check the deployment is returned
	deps, err := s.cli.AppsV1beta1().Deployments(s.namespace).List(metav1.ListOptions{})
	c.Assert(deps.Items, HasLen, 1)

	// Create a spec for the deployment
	depSpec, err := kube.CreateSpec(dep)
	c.Assert(err, IsNil)

	// Delete the deployment
	kubetest.DeleteDeployment(c, s.cli, s.namespace, dep)

	// Check the deployment is not returned
	deps, err = s.cli.AppsV1beta1().Deployments(s.namespace).List(metav1.ListOptions{})
	c.Assert(deps.Items, HasLen, 0)

	// Recreate the deployment from the spec
	newDep, err := kube.GetDeploymentFromSpec(depSpec)
	c.Assert(err, IsNil)
	newDep, err = s.cli.AppsV1beta1().Deployments(s.namespace).Create(newDep)
	c.Assert(err, IsNil)
	c.Assert(newDep.Name, Equals, dep.Name)

	// Check the deployment is returned
	deps, err = s.cli.AppsV1beta1().Deployments(s.namespace).List(metav1.ListOptions{})
	c.Assert(deps.Items, HasLen, 1)
}
