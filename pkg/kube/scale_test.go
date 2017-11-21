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

type ScaleSuite struct {
	cli       kubernetes.Interface
	namespace string
}

var _ = Suite(&ScaleSuite{})

func (s *ScaleSuite) SetUpSuite(c *C) {
	c.Skip("Too Slow")
	s.cli = kube.NewClient()
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kubescaletest-",
		},
	}
	cns, err := s.cli.Core().Namespaces().Create(ns)
	c.Assert(err, IsNil)
	s.namespace = cns.Name
}

func (s *ScaleSuite) TearDownSuite(c *C) {
	if s.namespace != "" {
		s.cli.Core().Namespaces().Delete(s.namespace, nil)
	}
}

const specFile = "./testdata/ss-1volume-fmt.yaml"

func (s *ScaleSuite) TestScaleStatefulSet(c *C) {
	// Create statefulset
	ctx := context.Background()
	ss := kubetest.CreateStatefulSetFromYamlSpec(ctx, c, s.cli, s.namespace, specFile)

	ss, err := s.cli.AppsV1beta1().StatefulSets(s.namespace).Get(ss.GetName(), metav1.GetOptions{})
	c.Assert(err, IsNil)

	vols := kube.StatefulSetVolumes(s.cli, ss)
	c.Assert(vols, HasLen, 1)

	r := *ss.Spec.Replicas
	for _, replicas := range []int32{r + 2, r} {
		err := kube.ScaleStatefulSet(ctx, s.cli, ss.GetNamespace(), ss.GetName(), replicas)
		c.Assert(err, IsNil)
		ss, err = s.cli.AppsV1beta1().StatefulSets(s.namespace).Get(ss.GetName(), metav1.GetOptions{})
		c.Assert(err, IsNil)
		c.Assert(*ss.Spec.Replicas, Equals, replicas)
		c.Assert(ss.Status.ReadyReplicas, Equals, replicas)

		vols := kube.StatefulSetVolumes(s.cli, ss)
		c.Assert(vols, HasLen, int(replicas))

		for pvc, _ := range vols {
			s.checkPVC(c, pvc, ss.Namespace)
		}
	}
}

func (s *ScaleSuite) checkPVC(c *C, name, namespace string) {
	_, err := s.cli.Core().PersistentVolumeClaims(namespace).Get(name, metav1.GetOptions{})
	c.Check(err, IsNil)
}
