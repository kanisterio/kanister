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
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/kanisterio/errkit"
	. "gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

type PodControllerTestSuite struct{}

var _ = Suite(&PodControllerTestSuite{})

const (
	podControllerNS      = "pod-runner-test"
	podControllerPodName = "test-pod"
)

func (s *PodControllerTestSuite) SetUpSuite(c *C) {
	err := os.Setenv("POD_NAMESPACE", podControllerNS)
	c.Assert(err, IsNil)
}

func (s *PodControllerTestSuite) TestPodControllerStartPod(c *C) {
	ctx := context.Background()
	cli := fake.NewSimpleClientset()

	simulatedError := errors.New("SimulatedError")

	cases := map[string]func(prp *FakePodControllerProcessor, pr PodController){
		"Pod creation failure": func(pcp *FakePodControllerProcessor, pc PodController) {
			pcp.CreatePodErr = simulatedError
			err := pc.StartPod(ctx)
			c.Assert(err, Not(IsNil))
			c.Assert(errors.Is(err, simulatedError), Equals, true)
			c.Assert(pcp.InCreatePodOptions, DeepEquals, &PodOptions{
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
			c.Assert(err, IsNil)
			c.Assert(pr.PodName(), Equals, podControllerPodName)
		},
		"Pod already created": func(prp *FakePodControllerProcessor, pr PodController) {
			prp.CreatePodRet = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: podControllerPodName,
				},
			}

			err := pr.StartPod(ctx)
			c.Assert(err, IsNil)

			prp.InCreatePodOptions = nil
			prp.CreatePodRet = nil
			prp.CreatePodErr = errors.New("CreatePod should not be invoked")

			err = pr.StartPod(ctx)
			c.Assert(err, Not(IsNil))
			c.Assert(errors.Is(err, ErrPodControllerPodAlreadyStarted), Equals, true)
			c.Assert(prp.InCreatePodOptions, IsNil)
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

func (s *PodControllerTestSuite) TestPodControllerWaitPod(c *C) {
	ctx := context.Background()
	cli := fake.NewSimpleClientset()

	simulatedError := errkit.Wrap(errors.New("SimulatedError"), "Wrapped")

	cases := map[string]func(pcp *FakePodControllerProcessor, pc PodController){
		"Waiting failed because pod not started yet": func(pcp *FakePodControllerProcessor, pc PodController) {
			err := pc.WaitForPodReady(ctx)
			c.Assert(err, Not(IsNil))
			c.Assert(errors.Is(err, ErrPodControllerPodNotStarted), Equals, true)
			c.Assert(pcp.InCreatePodOptions, IsNil)
		},
		"Waiting failed due to timeout": func(pcp *FakePodControllerProcessor, pc PodController) {
			pcp.CreatePodRet = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      podControllerPodName,
					Namespace: podControllerNS,
				},
			}
			err := pc.StartPod(ctx)
			c.Assert(err, IsNil)

			pcp.WaitForPodReadyErr = simulatedError
			err = pc.WaitForPodReady(ctx)
			c.Assert(err, Not(IsNil))
			c.Assert(pcp.InWaitForPodReadyPodName, Equals, podControllerPodName)
			c.Assert(pcp.InWaitForPodReadyNamespace, Equals, podControllerNS)
			c.Assert(errors.Is(err, pcp.WaitForPodReadyErr), Equals, true)

			c.Assert(err.Error(), Equals, fmt.Sprintf("Pod failed to become ready in time: %s", simulatedError.Error()))
			// Check that POD deletion was also invoked with expected arguments
		},
		"Waiting succeeded": func(pcp *FakePodControllerProcessor, pc PodController) {
			pcp.CreatePodRet = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: podControllerPodName,
				},
			}
			err := pc.StartPod(ctx)
			c.Assert(err, IsNil)
			err = pc.WaitForPodReady(ctx)
			c.Assert(err, IsNil)
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

func (s *PodControllerTestSuite) TestPodControllerStopPod(c *C) {
	ctx := context.Background()
	cli := fake.NewSimpleClientset()

	untouchedStr := "DEADBEEF"
	simulatedError := errors.New("SimulatedError")

	cases := map[string]func(pcp *FakePodControllerProcessor, pc PodController){
		"Pod not started yet": func(pcp *FakePodControllerProcessor, pc PodController) {
			err := pc.StopPod(ctx, 30*time.Second, int64(0))
			c.Assert(err, Not(IsNil))
			c.Assert(errors.Is(err, ErrPodControllerPodNotStarted), Equals, true)
			c.Assert(pcp.InDeletePodPodName, Equals, untouchedStr)
			c.Assert(pcp.InDeletePodNamespace, Equals, untouchedStr)
		},
		"Pod deletion error": func(pcp *FakePodControllerProcessor, pc PodController) {
			pcp.CreatePodRet = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      podControllerPodName,
					Namespace: podControllerNS,
				},
			}
			err := pc.StartPod(ctx)
			c.Assert(err, IsNil)

			pcp.DeletePodErr = simulatedError
			err = pc.StopPod(ctx, 30*time.Second, int64(0))
			c.Assert(err, Not(IsNil))
			c.Assert(errors.Is(err, simulatedError), Equals, true)
		},
		"Pod successfully deleted": func(pcp *FakePodControllerProcessor, pc PodController) {
			pcp.CreatePodRet = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      podControllerPodName,
					Namespace: podControllerNS,
				},
			}
			err := pc.StartPod(ctx)
			c.Assert(err, IsNil)

			err = pc.StopPod(ctx, 30*time.Second, int64(0))
			c.Assert(err, IsNil)
			c.Assert(pcp.InDeletePodPodName, Equals, podControllerPodName)
			c.Assert(pcp.InDeletePodNamespace, Equals, podControllerNS)
			gracePeriodSeconds := int64(0)
			c.Assert(pcp.InDeletePodOptions, DeepEquals, metav1.DeleteOptions{GracePeriodSeconds: &gracePeriodSeconds})
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

func (s *PodControllerTestSuite) TestPodControllerGetCommandExecutorAndFileWriter(c *C) {
	ctx := context.Background()
	cli := fake.NewSimpleClientset()

	cases := map[string]func(pcp *FakePodControllerProcessor, pc PodController){
		"Pod not started yet": func(_ *FakePodControllerProcessor, pc PodController) {
			pce, err := pc.GetCommandExecutor()
			c.Assert(pce, IsNil)
			c.Assert(err, Not(IsNil))
			c.Assert(errors.Is(err, ErrPodControllerPodNotStarted), Equals, true)

			pfw, err := pc.GetFileWriter()
			c.Assert(pfw, IsNil)
			c.Assert(err, Not(IsNil))
			c.Assert(errors.Is(err, ErrPodControllerPodNotStarted), Equals, true)
		},
		"Pod not ready yet": func(pcp *FakePodControllerProcessor, pc PodController) {
			pcp.CreatePodRet = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: podControllerPodName,
				},
			}
			err := pc.StartPod(ctx)
			c.Assert(err, IsNil)

			pce, err := pc.GetCommandExecutor()
			c.Assert(pce, IsNil)
			c.Assert(err, Not(IsNil))
			c.Assert(errors.Is(err, ErrPodControllerPodNotReady), Equals, true)

			pfw, err := pc.GetFileWriter()
			c.Assert(pfw, IsNil)
			c.Assert(err, Not(IsNil))
			c.Assert(errors.Is(err, ErrPodControllerPodNotReady), Equals, true)
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
			c.Assert(err, IsNil)

			err = pc.WaitForPodReady(ctx)
			c.Assert(err, IsNil)

			var epce PodCommandExecutor
			pce, err := pc.GetCommandExecutor()
			c.Assert(err, IsNil)
			c.Assert(pce, Implements, &epce)

			var epfw PodFileWriter
			pfw, err := pc.GetFileWriter()
			c.Assert(err, IsNil)
			c.Assert(pfw, Implements, &epfw)
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
