// Copyright 2019 The Kanister Authors.
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

// +build !unit

package kube

import (
	"bytes"
	"context"
	"strings"
	"time"

	. "gopkg.in/check.v1"
	v1 "k8s.io/api/core/v1"
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
	ctx := context.Background()
	var err error
	s.cli, err = NewClient()
	c.Assert(err, IsNil)
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "exectest-",
		},
	}
	ns, err = s.cli.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
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
	s.pod, err = s.cli.CoreV1().Pods(s.namespace).Create(ctx, pod, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	c.Assert(WaitForPodReady(ctxTimeout, s.cli, s.namespace, s.pod.Name), IsNil)
	s.pod, err = s.cli.CoreV1().Pods(s.namespace).Get(ctx, s.pod.Name, metav1.GetOptions{})
	c.Assert(err, IsNil)
}

func (s *ExecSuite) TearDownSuite(c *C) {
	if s.namespace != "" {
		err := s.cli.CoreV1().Namespaces().Delete(context.TODO(), s.namespace, metav1.DeleteOptions{})
		c.Assert(err, IsNil)
	}
}

func (s *ExecSuite) TestExecEcho(c *C) {
	cmd := []string{"sh", "-c", "cat -"}
	c.Assert(s.pod.Status.Phase, Equals, v1.PodRunning)
	c.Assert(len(s.pod.Status.ContainerStatuses) > 0, Equals, true)
	for _, cs := range s.pod.Status.ContainerStatuses {
		stdout, stderr, err := Exec(s.cli, s.pod.Namespace, s.pod.Name, cs.Name, cmd, bytes.NewBufferString("badabing"))
		c.Assert(err, IsNil)
		c.Assert(stdout, Equals, "badabing")
		c.Assert(stderr, Equals, "")
	}
}

func (s *ExecSuite) TestExecEchoDefaultContainer(c *C) {
	cmd := []string{"sh", "-c", "cat -"}
	c.Assert(s.pod.Status.Phase, Equals, v1.PodRunning)
	c.Assert(len(s.pod.Status.ContainerStatuses) > 0, Equals, true)
	stdout, stderr, err := Exec(s.cli, s.pod.Namespace, s.pod.Name, "", cmd, bytes.NewBufferString("badabing"))
	c.Assert(err, IsNil)
	c.Assert(stdout, Equals, "badabing")
	c.Assert(stderr, Equals, "")
}

func (s *ExecSuite) TestLSWithoutStdIn(c *C) {
	cmd := []string{"ls", "-l", "/home"}
	c.Assert(s.pod.Status.Phase, Equals, v1.PodRunning)
	c.Assert(len(s.pod.Status.ContainerStatuses) > 0, Equals, true)
	stdout, stderr, err := Exec(s.cli, s.pod.Namespace, s.pod.Name, "", cmd, nil)
	c.Assert(err, IsNil)
	c.Assert(stdout, Equals, "total 0")
	c.Assert(stderr, Equals, "")
}

func (s *ExecSuite) TestKopiaCommand(c *C) {
	ctx := context.Background()
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kopia-pod",
			Namespace: s.namespace,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				v1.Container{
					Name:  "kanister-sidecar",
					Image: "kanisterio/kanister-tools:0.37.0",
				},
			},
		},
	}
	p, err := s.cli.CoreV1().Pods(s.namespace).Create(ctx, pod, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	defer s.cli.CoreV1().Pods(s.namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
	ctxT, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	c.Assert(WaitForPodReady(ctxT, s.cli, s.namespace, p.Name), IsNil)
	// up until now below is how we were used to run kopia commands
	// "bash" "-c" "kopia repository create filesystem --path=$HOME/kopia_repo --password=newpass"
	// but now we don't want `bash -c`
	cmd := []string{"kopia", "repository", "create", "filesystem", "--path=$HOME/kopia_repo", "--password=newpass"}
	stdout, stderr, err := Exec(s.cli, pod.Namespace, pod.Name, "", cmd, nil)
	c.Assert(err, IsNil)
	c.Assert(strings.Contains(stdout, "Policy for (global):"), Equals, true)
	c.Assert(strings.Contains(stderr, "Initializing repository with:"), Equals, true)
}
