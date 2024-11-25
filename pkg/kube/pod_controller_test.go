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
	"os"
	"time"

	"github.com/kanisterio/errkit"
	"gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	cli := fake.NewSimpleClientset()

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
	cli := fake.NewSimpleClientset()

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
	cli := fake.NewSimpleClientset()

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
	cli := fake.NewSimpleClientset()

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
