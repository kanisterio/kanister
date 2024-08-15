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
	"errors"
	"strings"
	"time"

	. "gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type ExecSuite struct {
	cli       kubernetes.Interface
	namespace string
	pod       *corev1.Pod
}

var _ = Suite(&ExecSuite{})

func (s *ExecSuite) SetUpSuite(c *C) {
	ctx := context.Background()
	var err error
	s.cli, err = NewClient()
	c.Assert(err, IsNil)
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "exectest-",
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

func (s *ExecSuite) TestStderr(c *C) {
	cmd := []string{"sh", "-c", "echo -n hello >&2"}
	for _, cs := range s.pod.Status.ContainerStatuses {
		stdout, stderr, err := Exec(context.Background(), s.cli, s.pod.Namespace, s.pod.Name, cs.Name, cmd, nil)
		c.Assert(err, IsNil)
		c.Assert(stdout, Equals, "")
		c.Assert(stderr, Equals, "hello")
	}

	cmd = []string{"sh", "-c", "echo -n hello && exit 1"}
	for _, cs := range s.pod.Status.ContainerStatuses {
		stdout, stderr, err := Exec(context.Background(), s.cli, s.pod.Namespace, s.pod.Name, cs.Name, cmd, nil)
		c.Assert(err, NotNil)
		c.Assert(stdout, Equals, "hello")
		c.Assert(stderr, Equals, "")
	}

	cmd = []string{"sh", "-c", "count=0; while true; do printf $count; let count=$count+1; if [ $count -eq 6 ]; then exit 1; fi; done"}
	for _, cs := range s.pod.Status.ContainerStatuses {
		stdout, stderr, err := Exec(context.Background(), s.cli, s.pod.Namespace, s.pod.Name, cs.Name, cmd, nil)
		c.Assert(err, NotNil)
		c.Assert(stdout, Equals, "012345")
		c.Assert(stderr, Equals, "")
	}
}

func (s *ExecSuite) TestExecWithWriterOptions(c *C) {
	c.Assert(s.pod.Status.Phase, Equals, corev1.PodRunning)
	c.Assert(len(s.pod.Status.ContainerStatuses) > 0, Equals, true)

	var testCases = []struct {
		cmd         []string
		expectedOut string
		expectedErr string
	}{
		{
			cmd:         []string{"sh", "-c", "printf 'test'"},
			expectedOut: "test",
			expectedErr: "",
		},
		{
			cmd:         []string{"sh", "-c", "printf 'test' >&2"},
			expectedOut: "",
			expectedErr: "test",
		},
	}

	for _, testCase := range testCases {
		bufout := &bytes.Buffer{}
		buferr := &bytes.Buffer{}

		opts := ExecOptions{
			Command:       testCase.cmd,
			Namespace:     s.pod.Namespace,
			PodName:       s.pod.Name,
			ContainerName: "", // use default container
			Stdin:         nil,
			Stdout:        bufout,
			Stderr:        buferr,
		}
		err := ExecWithOptions(context.Background(), s.cli, opts)
		c.Assert(err, IsNil)
		c.Assert(bufout.String(), Equals, testCase.expectedOut)
		c.Assert(buferr.String(), Equals, testCase.expectedErr)
	}
}

func (s *ExecSuite) TestErrorInExecWithOptions(c *C) {
	c.Assert(s.pod.Status.Phase, Equals, corev1.PodRunning)
	c.Assert(len(s.pod.Status.ContainerStatuses) > 0, Equals, true)

	var testCases = []struct {
		cmd          []string
		expectedOut  []string
		expectedErr  []string
		expectedText string
	}{
		{
			cmd:          []string{"sh", "-c", "printf 'test\ntest1\ntest2\ntest3\ntest4\ntest5\ntest6\ntest7\ntest8\ntest9\ntest10' && exit 1"},
			expectedOut:  []string{"test", "test1", "test2", "test3", "test4", "test5", "test6", "test7", "test8", "test9", "test10"},
			expectedErr:  []string{},
			expectedText: "command terminated with exit code 1.\nstdout: test1\r\ntest2\r\ntest3\r\ntest4\r\ntest5\r\ntest6\r\ntest7\r\ntest8\r\ntest9\r\ntest10\nstderr: ",
		},
		{
			cmd:          []string{"sh", "-c", "printf 'test\ntest1\ntest2\ntest3\ntest4\ntest5\ntest6\ntest7\ntest8\ntest9\ntest10' >&2 && exit 1"},
			expectedOut:  []string{},
			expectedErr:  []string{"test", "test1", "test2", "test3", "test4", "test5", "test6", "test7", "test8", "test9", "test10"},
			expectedText: "command terminated with exit code 1.\nstdout: \nstderr: test1\r\ntest2\r\ntest3\r\ntest4\r\ntest5\r\ntest6\r\ntest7\r\ntest8\r\ntest9\r\ntest10",
		},
	}

	getSliceTail := func(slice []string, length int) []string {
		if len(slice) > length {
			return slice[len(slice)-length:]
		}

		return slice
	}

	for _, testCase := range testCases {
		// First invocation is without stdout and stderr buffers
		opts := ExecOptions{
			Command:       testCase.cmd,
			Namespace:     s.pod.Namespace,
			PodName:       s.pod.Name,
			ContainerName: "", // use default container
			Stdin:         nil,
		}
		err1 := ExecWithOptions(context.Background(), s.cli, opts)
		c.Assert(err1, Not(IsNil))

		var ee1 *ExecError
		ok := errors.As(err1, &ee1)
		c.Assert(ok, Equals, true)
		c.Assert(ee1.Stdout(), Not(Equals), testCase.expectedOut)
		c.Assert(ee1.Stderr(), Not(Equals), testCase.expectedErr)
		c.Assert(ee1.Error(), Equals, testCase.expectedText)

		// Now try the same with passing buffers for stdout and stderr
		// This should not affect returned error
		bufout := bytes.Buffer{}
		buferr := bytes.Buffer{}
		opts.Stdout = &bufout
		opts.Stderr = &buferr

		err2 := ExecWithOptions(context.Background(), s.cli, opts)
		c.Assert(err2, Not(IsNil))

		var ee2 *ExecError
		ok = errors.As(err2, &ee2)
		c.Assert(ok, Equals, true)

		// When error happens, stdout/stderr buffers should contain all lines produced by an app
		c.Assert(bufout.String(), Equals, strings.Join(testCase.expectedOut, "\n"))
		c.Assert(buferr.String(), Equals, strings.Join(testCase.expectedErr, "\n"))

		// When error happens, ExecError should contain only last ten lines of stdout/stderr
		c.Assert(ee2.Stdout(), Equals, strings.Join(getSliceTail(testCase.expectedOut, logTailDefaultLength), "\r\n"))
		c.Assert(ee2.Stderr(), Equals, strings.Join(getSliceTail(testCase.expectedErr, logTailDefaultLength), "\r\n"))

		// When error happens, ExecError should include stdout/stderr into its text representation
		c.Assert(ee2.Error(), Equals, testCase.expectedText)
	}
}

func (s *ExecSuite) TestExecEcho(c *C) {
	cmd := []string{"sh", "-c", "cat -"}
	c.Assert(s.pod.Status.Phase, Equals, corev1.PodRunning)
	c.Assert(len(s.pod.Status.ContainerStatuses) > 0, Equals, true)
	for _, cs := range s.pod.Status.ContainerStatuses {
		stdout, stderr, err := Exec(context.Background(), s.cli, s.pod.Namespace, s.pod.Name, cs.Name, cmd, bytes.NewBufferString("badabing"))
		c.Assert(err, IsNil)
		c.Assert(stdout, Equals, "badabing")
		c.Assert(stderr, Equals, "")
	}
}

func (s *ExecSuite) TestExecEchoDefaultContainer(c *C) {
	cmd := []string{"sh", "-c", "cat -"}
	c.Assert(s.pod.Status.Phase, Equals, corev1.PodRunning)
	c.Assert(len(s.pod.Status.ContainerStatuses) > 0, Equals, true)
	stdout, stderr, err := Exec(context.Background(), s.cli, s.pod.Namespace, s.pod.Name, "", cmd, bytes.NewBufferString("badabing"))
	c.Assert(err, IsNil)
	c.Assert(stdout, Equals, "badabing")
	c.Assert(stderr, Equals, "")
}

func (s *ExecSuite) TestLSWithoutStdIn(c *C) {
	cmd := []string{"ls", "-l", "/home"}
	c.Assert(s.pod.Status.Phase, Equals, corev1.PodRunning)
	c.Assert(len(s.pod.Status.ContainerStatuses) > 0, Equals, true)
	stdout, stderr, err := Exec(context.Background(), s.cli, s.pod.Namespace, s.pod.Name, "", cmd, nil)
	c.Assert(err, IsNil)
	c.Assert(stdout, Equals, "total 0")
	c.Assert(stderr, Equals, "")
}

func (s *ExecSuite) TestKopiaCommand(c *C) {
	ctx := context.Background()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kopia-pod",
			Namespace: s.namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "kanister-sidecar",
					Image: "ghcr.io/kanisterio/kanister-tools:0.37.0",
				},
			},
		},
	}
	p, err := s.cli.CoreV1().Pods(s.namespace).Create(ctx, pod, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	defer func() {
		err := s.cli.CoreV1().Pods(s.namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
		c.Assert(err, IsNil)
	}()
	ctxT, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	c.Assert(WaitForPodReady(ctxT, s.cli, s.namespace, p.Name), IsNil)
	// up until now below is how we were used to run kopia commands
	// "bash" "-c" "kopia repository create filesystem --path=$HOME/kopia_repo --password=newpass"
	// but now we don't want `bash -c`
	cmd := []string{"kopia", "repository", "create", "filesystem", "--path=$HOME/kopia_repo", "--password=newpass"}
	stdout, stderr, err := Exec(context.Background(), s.cli, pod.Namespace, pod.Name, "", cmd, nil)
	c.Assert(err, IsNil)
	c.Assert(strings.Contains(stdout, "Policy for (global):"), Equals, true)
	c.Assert(strings.Contains(stderr, "Initializing repository with:"), Equals, true)
}

// TestContextTimeout verifies that when context is cancelled during command execution,
// execution will be interrupted and proper error will be returned. The stdout, stderr streams should be captured.
func (s *ExecSuite) TestContextTimeout(c *C) {
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()
	cmd := []string{"sh", "-c", "echo abc && sleep 2 && echo def"}
	for _, cs := range s.pod.Status.ContainerStatuses {
		stdout, stderr, err := Exec(ctx, s.cli, s.pod.Namespace, s.pod.Name, cs.Name, cmd, nil)
		c.Assert(err, NotNil)
		c.Assert(stdout, Equals, "abc")
		c.Assert(stderr, Equals, "")
		c.Assert(err.Error(), Equals, "Failed to exec command in pod: context deadline exceeded.\nstdout: abc\nstderr: ")
	}
}

// TestCancelledContext verifies that when execution is proceeded with context which is already cancelled,
// proper error will be returned. The stdout, stderr streams should remain empty, because command has not been executed.
func (s *ExecSuite) TestCancelledContext(c *C) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cmd := []string{"sh", "-c", "echo abc && sleep 2"}
	for _, cs := range s.pod.Status.ContainerStatuses {
		stdout, stderr, err := Exec(ctx, s.cli, s.pod.Namespace, s.pod.Name, cs.Name, cmd, nil)
		c.Assert(err, NotNil)
		c.Assert(stdout, Equals, "")
		c.Assert(stderr, Equals, "")
		c.Assert(err.Error(), Matches, "Failed to exec command in pod: error sending request: Post \".*\": .*: operation was canceled.\nstdout: \nstderr: ")
	}
}
