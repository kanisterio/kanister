// +build !unit

package kube

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "gopkg.in/check.v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
)

type PodSuite struct {
	cli       kubernetes.Interface
	namespace string
}

var _ = Suite(&PodSuite{})

func (s *PodSuite) SetUpSuite(c *C) {
	var err error
	s.cli, err = NewClient()
	c.Assert(err, IsNil)
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "podtest-",
		},
	}
	ns, err = s.cli.Core().Namespaces().Create(ns)
	c.Assert(err, IsNil)
	s.namespace = ns.Name
}

func (s *PodSuite) TearDownSuite(c *C) {
	if s.namespace != "" {
		err := s.cli.Core().Namespaces().Delete(s.namespace, nil)
		c.Assert(err, IsNil)
	}
}

func (s *PodSuite) TestPod(c *C) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	pod, err := CreatePod(context.Background(), s.cli, &PodOptions{
		Namespace:    s.namespace,
		GenerateName: "test-",
		Image:        "kanisterio/kanister-tools:0.14.0",
		Command:      []string{"sh", "-c", "tail -f /dev/null"},
	})
	c.Assert(err, IsNil)
	c.Assert(WaitForPodReady(ctx, s.cli, s.namespace, pod.Name), IsNil)
	c.Assert(DeletePod(context.Background(), s.cli, pod), IsNil)
}

func (s *PodSuite) TestPodWithVolumes(c *C) {
	cli := fake.NewSimpleClientset()
	vols := map[string]string{"pvc-test": "/mnt/data1"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	var p *v1.Pod
	cli.PrependReactor("create", "pods", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		fmt.Println("found pod")
		ca := action.(testing.CreateAction)
		p = ca.GetObject().(*v1.Pod)
		return false, nil, nil
	})
	cli.PrependReactor("get", "pods", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		p.Status.Phase = v1.PodRunning
		return true, p, nil
	})
	pod, err := CreatePod(ctx, cli, &PodOptions{
		Namespace:    s.namespace,
		GenerateName: "test-",
		Image:        "kanisterio/kanister-tools:0.14.0",
		Command:      []string{"sh", "-c", "tail -f /dev/null"},
		Volumes:      vols,
	})
	c.Assert(err, IsNil)
	c.Assert(pod.Spec.Volumes, HasLen, 1)
	c.Assert(pod.Spec.Volumes[0].VolumeSource.PersistentVolumeClaim.ClaimName, Equals, "pvc-test")
	c.Assert(pod.Spec.Containers[0].VolumeMounts[0].MountPath, Equals, "/mnt/data1")
}

func (s *PodSuite) TestGetPodLogs(c *C) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	pod, err := CreatePod(context.Background(), s.cli, &PodOptions{
		Namespace:    s.namespace,
		GenerateName: "test-",
		Image:        "kanisterio/kanister-tools:0.14.0",
		Command:      []string{"sh", "-c", "echo hello"},
	})
	c.Assert(err, IsNil)
	c.Assert(WaitForPodCompletion(ctx, s.cli, s.namespace, pod.Name), IsNil)
	logs, err := GetPodLogs(ctx, s.cli, s.namespace, pod.Name)
	c.Assert(err, IsNil)
	c.Assert(strings.Contains(logs, "hello"), Equals, true)
	c.Assert(DeletePod(context.Background(), s.cli, pod), IsNil)
}
