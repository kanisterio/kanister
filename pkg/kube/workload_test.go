// +build !unit

package kube_test

import (
	. "gopkg.in/check.v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/kube"
	kubetest "github.com/kanisterio/kanister/pkg/kube/test"
)

type WorkloadSuite struct {
	cli       kubernetes.Interface
	namespace string
}

var _ = Suite(&WorkloadSuite{})

func (s *WorkloadSuite) SetUpSuite(c *C) {
	c.Skip("Too slow")
	s.cli = kube.NewClient()
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kubeworkloadtest-",
		},
	}
	cns, err := s.cli.Core().Namespaces().Create(ns)
	c.Assert(err, IsNil)
	s.namespace = cns.Name
}

func (s *WorkloadSuite) TearDownSuite(c *C) {
	if s.namespace != "" {
		s.cli.Core().Namespaces().Delete(s.namespace, nil)
	}
}

func (s *WorkloadSuite) TestDeployment(c *C) {
	numVols := 2
	// Create deployment
	deployment := kubetest.CreateDeployment(c, s.cli, s.namespace, map[string]string{"d": "1"}, map[string]string{}, numVols)

	vols := kube.DeploymentVolumes(s.cli, deployment)
	c.Assert(vols, HasLen, numVols)
	for pvc, _ := range vols {
		s.checkPVC(c, pvc, deployment.Namespace)
	}
}
func (s *WorkloadSuite) checkPVC(c *C, name, namespace string) {
	_, err := s.cli.Core().PersistentVolumeClaims(namespace).Get(name, metav1.GetOptions{})
	c.Check(err, IsNil)
}
