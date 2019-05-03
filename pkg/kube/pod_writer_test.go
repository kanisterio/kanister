// +build !unit

package kube

import (
	"bytes"
	"context"
	"time"

	. "gopkg.in/check.v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type PodWriteSuite struct {
	cli       kubernetes.Interface
	namespace string
	pod       *v1.Pod
}

var _ = Suite(&PodWriteSuite{})

func (p *PodWriteSuite) SetUpSuite(c *C) {
	var err error
	p.cli, err = NewClient()
	c.Assert(err, IsNil)
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "podwritertest-",
		},
	}
	ns, err = p.cli.Core().Namespaces().Create(ns)
	c.Assert(err, IsNil)
	p.namespace = ns.Name
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
	p.pod, err = p.cli.Core().Pods(p.namespace).Create(pod)
	c.Assert(err, IsNil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	c.Assert(WaitForPodReady(ctx, p.cli, p.namespace, p.pod.Name), IsNil)
	p.pod, err = p.cli.Core().Pods(p.namespace).Get(p.pod.Name, metav1.GetOptions{})
	c.Assert(err, IsNil)
}

func (p *PodWriteSuite) TearDownSuite(c *C) {
	if p.namespace != "" {
		err := p.cli.Core().Namespaces().Delete(p.namespace, nil)
		c.Assert(err, IsNil)
	}
}
func (p *PodWriteSuite) TestPodWriter(c *C) {
	path := "/tmp/test.txt"
	c.Assert(p.pod.Status.Phase, Equals, v1.PodRunning)
	c.Assert(len(p.pod.Status.ContainerStatuses) > 0, Equals, true)
	for _, cs := range p.pod.Status.ContainerStatuses {
		pw := NewPodWriter(p.cli, path, bytes.NewBufferString("badabing"))
		err := pw.Write(context.Background(), p.pod.Namespace, p.pod.Name, cs.Name)
		c.Assert(err, IsNil)
		cmd := []string{"sh", "-c", "cat " + pw.path}
		stdout, stderr, err := Exec(p.cli, p.pod.Namespace, p.pod.Name, cs.Name, cmd, nil)
		c.Assert(err, IsNil)
		c.Assert(stdout, Equals, "badabing")
		c.Assert(stderr, Equals, "")
		err = pw.Remove(context.Background(), p.pod.Namespace, p.pod.Name, cs.Name)
		c.Assert(err, IsNil)
	}
}
