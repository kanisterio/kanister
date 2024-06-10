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

//go:build !unit
// +build !unit

package kube

import (
	"bytes"
	"context"
	"path/filepath"
	"time"

	. "gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type PodWriteSuite struct {
	cli       kubernetes.Interface
	namespace string
	pod       *corev1.Pod
}

var _ = Suite(&PodWriteSuite{})

func (p *PodWriteSuite) SetUpSuite(c *C) {
	var err error
	ctx := context.Background()
	p.cli, err = NewClient()
	c.Assert(err, IsNil)
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "podwritertest-",
		},
	}
	ns, err = p.cli.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	p.namespace = ns.Name
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "testpod"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "testcontainer",
					Image:   "busybox",
					Command: []string{"sh", "-c", "tail -f /dev/null"},
				},
			},
		},
	}
	p.pod, err = p.cli.CoreV1().Pods(p.namespace).Create(ctx, pod, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	c.Assert(WaitForPodReady(ctx, p.cli, p.namespace, p.pod.Name), IsNil)
	p.pod, err = p.cli.CoreV1().Pods(p.namespace).Get(ctx, p.pod.Name, metav1.GetOptions{})
	c.Assert(err, IsNil)
}

func (p *PodWriteSuite) TearDownSuite(c *C) {
	if p.namespace != "" {
		err := p.cli.CoreV1().Namespaces().Delete(context.TODO(), p.namespace, metav1.DeleteOptions{})
		c.Assert(err, IsNil)
	}
}
func (p *PodWriteSuite) TestPodWriter(c *C) {
	path := "/tmp/test.txt"
	c.Assert(p.pod.Status.Phase, Equals, corev1.PodRunning)
	c.Assert(len(p.pod.Status.ContainerStatuses) > 0, Equals, true)
	for _, cs := range p.pod.Status.ContainerStatuses {
		pw := NewPodWriter(p.cli, path, bytes.NewBufferString("badabing"))
		err := pw.Write(context.Background(), p.pod.Namespace, p.pod.Name, cs.Name)
		c.Assert(err, IsNil)
		cmd := []string{"sh", "-c", "cat " + filepath.Clean(path)}
		stdout, stderr, err := Exec(context.Background(), p.cli, p.pod.Namespace, p.pod.Name, cs.Name, cmd, nil)
		c.Assert(err, IsNil)
		c.Assert(stdout, Equals, "badabing")
		c.Assert(stderr, Equals, "")
		err = pw.Remove(context.Background(), p.pod.Namespace, p.pod.Name, cs.Name)
		c.Assert(err, IsNil)
	}
}
