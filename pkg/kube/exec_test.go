// +build !unit

package kube

import (
	. "gopkg.in/check.v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type ExecSuite struct {
	cli       kubernetes.Interface
	namespace string
	pod       *v1.Pod
}

var _ = Suite(&ExecSuite{})

func (s *ExecSuite) SetUpSuite(c *C) {
	var err error
	s.cli, err = NewClient()
	c.Assert(err, IsNil)
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "exectest-",
		},
	}
	ns, err = s.cli.Core().Namespaces().Create(ns)
	c.Assert(err, IsNil)
	s.namespace = ns.Name
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "testpod"},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				v1.Container{
					Name:    "testcontainer",
					Image:   "busybox",
					Command: []string{"sh", "-c", "tail -f /dev/null"},
				},
			},
		},
	}
	s.pod, err = s.cli.Core().Pods(s.namespace).Create(pod)
	c.Assert(err, IsNil)
}

func (s *ExecSuite) TearDownSuite(c *C) {
	if s.namespace != "" {
		err := s.cli.Core().Namespaces().Delete(s.namespace, nil)
		c.Assert(err, IsNil)
	}
}
func (s *ExecSuite) TestExecEcho(c *C) {
	cmd := []string{"sh", "-c", "echo badabing"}
	for _, cs := range s.pod.Status.ContainerStatuses {
		stdout, stderr, err := Exec(s.cli, s.pod.Namespace, s.pod.Name, cs.Name, cmd)
		c.Assert(err, IsNil)
		c.Assert(stdout, Equals, "badabing")
		c.Assert(stderr, Equals, "")
	}
}
