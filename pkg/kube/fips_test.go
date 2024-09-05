// Copyright 2022 The Kanister Authors.
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
	"bufio"
	"context"
	"strings"
	"time"

	. "gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type FIPSSuite struct {
	cli       kubernetes.Interface
	namespace string
	pod       *corev1.Pod
}

var _ = Suite(&FIPSSuite{})

func (s *FIPSSuite) SetUpSuite(c *C) {
	ctx := context.Background()
	var err error
	s.cli, err = NewClient()
	c.Assert(err, IsNil)
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "fipstest-",
		},
	}
	ns, err = s.cli.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	s.namespace = ns.Name
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "testpod"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "testcontainer",
					Image:   "ghcr.io/kanisterio/kanister-tools:v9.99.9-dev",
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
	// install go in kanister-tools pod
	cmd := []string{"microdnf", "install", "-y", "go"}
	for _, cs := range s.pod.Status.ContainerStatuses {
		stdout, stderr, err := Exec(ctx, s.cli, s.pod.Namespace, s.pod.Name, cs.Name, cmd, nil)
		c.Assert(err, IsNil)
		c.Log(stderr)
		c.Log(stdout)
	}
}

func (s *FIPSSuite) TearDownSuite(c *C) {
	if s.namespace != "" {
		err := s.cli.CoreV1().Namespaces().Delete(context.TODO(), s.namespace, metav1.DeleteOptions{})
		c.Assert(err, IsNil)
	}
}

func (s *FIPSSuite) TestFIPSBoringEnabled(c *C) {
	for _, tool := range []string{
		"/usr/local/bin/kopia",
		"/usr/local/bin/kando",
	} {
		c.Logf("Testing %s", tool)
		cmd := []string{"go", "tool", "nm", tool}
		for _, cs := range s.pod.Status.ContainerStatuses {
			stdout, stderr, err := Exec(context.Background(), s.cli, s.pod.Namespace, s.pod.Name, cs.Name, cmd, nil)
			c.Assert(err, IsNil)
			c.Assert(stderr, Equals, "")
			scanner := bufio.NewScanner(strings.NewReader(stdout))
			fipsModeSet := false
			for scanner.Scan() {
				if strings.Contains(scanner.Text(), "FIPS") {
					c.Log(scanner.Text())
				}
				if strings.Contains(scanner.Text(), "FIPS_mode_set") {
					fipsModeSet = true
				}
			}
			c.Assert(fipsModeSet, Equals, true)
		}
	}
}
