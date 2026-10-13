// Copyright 2023 The Kanister Authors.
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

package kube

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/kanisterio/errkit"
	"gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

type PodControllerTestSuite struct{}

var _ = check.Suite(&PodControllerTestSuite{})

const (
	podControllerNS      = "pod-runner-test"
	podControllerPodName = "test-pod"
)

func (s *PodControllerTestSuite) SetUpSuite(c *check.C) {
	err := os.Setenv("POD_NAMESPACE", podControllerNS)
	c.Assert(err, check.IsNil)
}

func (s *PodControllerTestSuite) TestPodControllerStartPod(c *check.C) {
	ctx := context.Background()
	cli := fake.NewClientset()

	simulatedError := errkit.New("SimulatedError")

	cases := map[string]func(prp *FakePodControllerProcessor, pr PodController){
		"Pod creation failure": func(pcp *FakePodControllerProcessor, pc PodController) {
			pcp.CreatePodErr = simulatedError
			err := pc.StartPod(ctx)
			c.Assert(err, check.Not(check.IsNil))
			c.Assert(errkit.Is(err, simulatedError), check.Equals, true)
			c.Assert(pcp.InCreatePodOptions, check.DeepEquals, &PodOptions{
				Namespace: podControllerNS,
				Name:      podControllerPodName,
			})
		},
		"Pod successfully started": func(prp *FakePodControllerProcessor, pr PodController) {
			prp.CreatePodRet = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: podControllerPodName,
				},
			}
			err := pr.StartPod(ctx)
			c.Assert(err, check.IsNil)
			c.Assert(pr.PodName(), check.Equals, podControllerPodName)
		},
		"Pod already created": func(prp *FakePodControllerProcessor, pr PodController) {
			prp.CreatePodRet = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: podControllerPodName,
				},
			}

			err := pr.StartPod(ctx)
			c.Assert(err, check.IsNil)

			prp.InCreatePodOptions = nil
			prp.CreatePodRet = nil
			prp.CreatePodErr = errkit.New("CreatePod should not be invoked")

			err = pr.StartPod(ctx)
			c.Assert(err, check.Not(check.IsNil))
			c.Assert(errkit.Is(err, ErrPodControllerPodAlreadyStarted), check.Equals, true)
			c.Assert(prp.InCreatePodOptions, check.IsNil)
		},
	}

	for l, tc := range cases {
		c.Log(l)

		pcp := &FakePodControllerProcessor{}

		pc := NewPodController(cli, &PodOptions{
			Namespace: podControllerNS,
			Name:      podControllerPodName,
		}, WithPodControllerProcessor(pcp))

		tc(pcp, pc)
	}
}

func (s *PodControllerTestSuite) TestPodControllerWaitPod(c *check.C) {
	ctx := context.Background()
	cli := fake.NewClientset()

	simulatedError := errkit.Wrap(errkit.New("SimulatedError"), "Wrapped")

	cases := map[string]func(pcp *FakePodControllerProcessor, pc PodController){
		"Waiting failed because pod not started yet": func(pcp *FakePodControllerProcessor, pc PodController) {
			err := pc.WaitForPodReady(ctx)
			c.Assert(err, check.Not(check.IsNil))
			c.Assert(errkit.Is(err, ErrPodControllerPodNotStarted), check.Equals, true)
			c.Assert(pcp.InCreatePodOptions, check.IsNil)
		},
		"Waiting failed due to timeout": func(pcp *FakePodControllerProcessor, pc PodController) {
			pcp.CreatePodRet = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      podControllerPodName,
					Namespace: podControllerNS,
				},
			}
			err := pc.StartPod(ctx)
			c.Assert(err, check.IsNil)

			pcp.WaitForPodReadyErr = simulatedError
			err = pc.WaitForPodReady(ctx)
			c.Assert(err, check.Not(check.IsNil))
			c.Assert(pcp.InWaitForPodReadyPodName, check.Equals, podControllerPodName)
			c.Assert(pcp.InWaitForPodReadyNamespace, check.Equals, podControllerNS)
			c.Assert(errkit.Is(err, pcp.WaitForPodReadyErr), check.Equals, true)

			c.Assert(err.Error(), check.Equals, fmt.Sprintf("Pod failed to become ready in time: %s", simulatedError.Error()))
			// Check that POD deletion was also invoked with expected arguments
		},
		"Waiting succeeded": func(pcp *FakePodControllerProcessor, pc PodController) {
			pcp.CreatePodRet = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: podControllerPodName,
				},
			}
			err := pc.StartPod(ctx)
			c.Assert(err, check.IsNil)
			err = pc.WaitForPodReady(ctx)
			c.Assert(err, check.IsNil)
		},
	}

	for l, tc := range cases {
		c.Log(l)

		pcp := &FakePodControllerProcessor{}

		pc := NewPodController(cli, &PodOptions{
			Namespace: podControllerNS,
			Name:      podControllerPodName,
		}, WithPodControllerProcessor(pcp))

		tc(pcp, pc)
	}
}

func (s *PodControllerTestSuite) TestPodControllerStopPod(c *check.C) {
	ctx := context.Background()
	cli := fake.NewClientset()

	untouchedStr := "DEADBEEF"
	simulatedError := errkit.New("SimulatedError")

	cases := map[string]func(pcp *FakePodControllerProcessor, pc PodController){
		"Pod not started yet": func(pcp *FakePodControllerProcessor, pc PodController) {
			err := pc.StopPod(ctx, 30*time.Second, int64(0))
			c.Assert(err, check.Not(check.IsNil))
			c.Assert(errkit.Is(err, ErrPodControllerPodNotStarted), check.Equals, true)
			c.Assert(pcp.InDeletePodPodName, check.Equals, untouchedStr)
			c.Assert(pcp.InDeletePodNamespace, check.Equals, untouchedStr)
		},
		"Pod deletion error": func(pcp *FakePodControllerProcessor, pc PodController) {
			pcp.CreatePodRet = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      podControllerPodName,
					Namespace: podControllerNS,
				},
			}
			err := pc.StartPod(ctx)
			c.Assert(err, check.IsNil)

			pcp.DeletePodErr = simulatedError
			err = pc.StopPod(ctx, 30*time.Second, int64(0))
			c.Assert(err, check.Not(check.IsNil))
			c.Assert(errkit.Is(err, simulatedError), check.Equals, true)
		},
		"Pod successfully deleted": func(pcp *FakePodControllerProcessor, pc PodController) {
			pcp.CreatePodRet = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      podControllerPodName,
					Namespace: podControllerNS,
				},
			}
			err := pc.StartPod(ctx)
			c.Assert(err, check.IsNil)

			err = pc.StopPod(ctx, 30*time.Second, int64(0))
			c.Assert(err, check.IsNil)
			c.Assert(pcp.InDeletePodPodName, check.Equals, podControllerPodName)
			c.Assert(pcp.InDeletePodNamespace, check.Equals, podControllerNS)
			gracePeriodSeconds := int64(0)
			c.Assert(pcp.InDeletePodOptions, check.DeepEquals, metav1.DeleteOptions{GracePeriodSeconds: &gracePeriodSeconds})
		},
	}

	for l, tc := range cases {
		c.Log(l)

		pcp := &FakePodControllerProcessor{
			InDeletePodNamespace: untouchedStr,
			InDeletePodPodName:   untouchedStr,
			InDeletePodOptions:   metav1.DeleteOptions{},
		}

		pc := NewPodController(cli, &PodOptions{
			Name: podControllerPodName,
		}, WithPodControllerProcessor(pcp))

		tc(pcp, pc)
	}
}

func (s *PodControllerTestSuite) TestPodControllerGetCommandExecutorAndFileWriter(c *check.C) {
	ctx := context.Background()
	cli := fake.NewClientset()

	cases := map[string]func(pcp *FakePodControllerProcessor, pc PodController){
		"Pod not started yet": func(_ *FakePodControllerProcessor, pc PodController) {
			pce, err := pc.GetCommandExecutor()
			c.Assert(pce, check.IsNil)
			c.Assert(err, check.Not(check.IsNil))
			c.Assert(errkit.Is(err, ErrPodControllerPodNotStarted), check.Equals, true)

			pfw, err := pc.GetFileWriter()
			c.Assert(pfw, check.IsNil)
			c.Assert(err, check.Not(check.IsNil))
			c.Assert(errkit.Is(err, ErrPodControllerPodNotStarted), check.Equals, true)
		},
		"Pod not ready yet": func(pcp *FakePodControllerProcessor, pc PodController) {
			pcp.CreatePodRet = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: podControllerPodName,
				},
			}
			err := pc.StartPod(ctx)
			c.Assert(err, check.IsNil)

			pce, err := pc.GetCommandExecutor()
			c.Assert(pce, check.IsNil)
			c.Assert(err, check.Not(check.IsNil))
			c.Assert(errkit.Is(err, ErrPodControllerPodNotReady), check.Equals, true)

			pfw, err := pc.GetFileWriter()
			c.Assert(pfw, check.IsNil)
			c.Assert(err, check.Not(check.IsNil))
			c.Assert(errkit.Is(err, ErrPodControllerPodNotReady), check.Equals, true)
		},
		"CommandExecutor successfully returned": func(pcp *FakePodControllerProcessor, pc PodController) {
			pcp.CreatePodRet = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: podControllerPodName,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "some-test-pod"},
					},
				},
			}
			err := pc.StartPod(ctx)
			c.Assert(err, check.IsNil)

			err = pc.WaitForPodReady(ctx)
			c.Assert(err, check.IsNil)

			var epce PodCommandExecutor
			pce, err := pc.GetCommandExecutor()
			c.Assert(err, check.IsNil)
			c.Assert(pce, check.Implements, &epce)

			var epfw PodFileWriter
			pfw, err := pc.GetFileWriter()
			c.Assert(err, check.IsNil)
			c.Assert(pfw, check.Implements, &epfw)
		},
	}

	for l, tc := range cases {
		c.Log(l)

		pcp := &FakePodControllerProcessor{}

		pc := NewPodController(cli, &PodOptions{
			Namespace: podControllerNS,
			Name:      podControllerPodName,
		}, WithPodControllerProcessor(pcp))

		tc(pcp, pc)
	}
}

// RestoreLogStreamReaderTestSuite covers the nil-dereference fix in
// restoreLogStreamReader.
type RestoreLogStreamReaderTestSuite struct{}

var _ = check.Suite(&RestoreLogStreamReaderTestSuite{})

// eofReadCloser is a mock io.ReadCloser that returns io.EOF on Read.
type eofReadCloser struct{ closed bool }

func (e *eofReadCloser) Read([]byte) (int, error) { return 0, io.EOF }
func (e *eofReadCloser) Close() error             { e.closed = true; return nil }

// TestCloseWithNilReader verifies that Close() does not panic when s.reader is nil.
func (s *RestoreLogStreamReaderTestSuite) TestCloseWithNilReader(c *check.C) {
	r := &restoreLogStreamReader{reader: nil}
	err := r.Close()
	c.Assert(err, check.IsNil)
}

// TestReadReturnsErrorWhenStreamFuncFails verifies the full failure path:
// - mock reader returns io.EOF (kubelet 4-hour timeout)
// - pod is still Running so we attempt to re-establish the stream
// - streamFunc returns an error (simulating pod termination mid-stream)
// Read() must return the error cleanly and Close() must not panic afterwards.
func (s *RestoreLogStreamReaderTestSuite) TestReadReturnsErrorWhenStreamFuncFails(c *check.C) {
	const (
		ns        = "test-ns"
		podName   = "test-pod"
		container = "test-container"
	)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: podName, Namespace: ns},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}
	cli := fake.NewClientset(pod)

	streamErr := errkit.New("pod terminating")
	mockReader := &eofReadCloser{}

	r := &restoreLogStreamReader{
		ctx:           context.Background(),
		cli:           cli,
		namespace:     ns,
		podName:       podName,
		containerName: container,
		reader:        mockReader,
		streamFunc: func(_ context.Context, _ kubernetes.Interface, _, _, _ string, _ *metav1.Time) (io.ReadCloser, error) {
			return nil, streamErr
		},
	}

	_, err := r.Read(make([]byte, 32))
	c.Assert(err, check.Equals, streamErr)

	// s.reader must still be the closed (non-nil) mock — Close() must not panic.
	c.Assert(r.reader, check.NotNil)
	err = r.Close()
	c.Assert(err, check.IsNil)
	c.Assert(mockReader.closed, check.Equals, true)
}
