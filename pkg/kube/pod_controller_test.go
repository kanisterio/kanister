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

	"github.com/pkg/errors"
	. "gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

type PodControllerTestSuite struct{}

var _ = Suite(&PodControllerTestSuite{})

const (
	podControllerNS      = "pod-runner-test"
	podControllerPodName = "test-pod"
)

func (s *PodControllerTestSuite) SetUpSuite(c *C) {
	os.Setenv("POD_NAMESPACE", podControllerNS)
}

type fakePodControllerProcessor struct {
	inWaitForPodReadyNamespace string
	inWaitForPodReadyPodName   string
	waitForPodReadyErr         error

	inWaitForPodCompletionNamespace string
	inWaitForPodCompletionPodName   string
	waitForPodCompletionErr         error

	inDeletePodNamespace string
	inDeletePodPodName   string
	inDeletePodOptions   metav1.DeleteOptions
	deletePodErr         error

	inCreatePodCli     kubernetes.Interface
	inCreatePodOptions *PodOptions
	createPodRet       *corev1.Pod
	createPodErr       error
}

func (f *fakePodControllerProcessor) createPod(_ context.Context, cli kubernetes.Interface, options *PodOptions) (*corev1.Pod, error) {
	f.inCreatePodCli = cli
	f.inCreatePodOptions = options
	return f.createPodRet, f.createPodErr
}

func (f *fakePodControllerProcessor) waitForPodCompletion(ctx context.Context, namespace, podName string) error {
	f.inWaitForPodCompletionNamespace = namespace
	f.inWaitForPodCompletionPodName = podName
	return f.waitForPodCompletionErr
}

func (f *fakePodControllerProcessor) waitForPodReady(ctx context.Context, namespace, podName string) error {
	f.inWaitForPodReadyPodName = podName
	f.inWaitForPodReadyNamespace = namespace
	return f.waitForPodReadyErr
}

func (f *fakePodControllerProcessor) deletePod(_ context.Context, namespace string, podName string, opts metav1.DeleteOptions) error {
	f.inDeletePodNamespace = namespace
	f.inDeletePodPodName = podName
	f.inDeletePodOptions = opts

	return f.deletePodErr
}

func (s *PodControllerTestSuite) TestPodControllerStartPod(c *C) {
	ctx := context.Background()
	cli := fake.NewSimpleClientset()

	simulatedError := errors.New("SimulatedError")

	cases := map[string]func(prp *fakePodControllerProcessor, pr PodController){
		"Pod creation failure": func(pcp *fakePodControllerProcessor, pc PodController) {
			pcp.createPodErr = simulatedError
			err := pc.StartPod(ctx)
			c.Assert(err, Not(IsNil))
			c.Assert(errors.Is(err, simulatedError), Equals, true)
			c.Assert(pcp.inCreatePodCli, Equals, cli)
			c.Assert(pcp.inCreatePodOptions, DeepEquals, &PodOptions{
				Namespace: podControllerNS,
				Name:      podControllerPodName,
			})
		},
		"Pod successfully started": func(prp *fakePodControllerProcessor, pr PodController) {
			prp.createPodRet = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: podControllerPodName,
				},
			}
			err := pr.StartPod(ctx)
			c.Assert(err, IsNil)
			c.Assert(pr.PodName(), Equals, podControllerPodName)
		},
		"Pod already created": func(prp *fakePodControllerProcessor, pr PodController) {
			prp.createPodRet = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: podControllerPodName,
				},
			}

			err := pr.StartPod(ctx)
			c.Assert(err, IsNil)
			c.Assert(prp.inCreatePodCli, Equals, cli)

			prp.inCreatePodCli = nil
			prp.inCreatePodOptions = nil
			prp.createPodRet = nil
			prp.createPodErr = errors.New("CreatePod should not be invoked")

			err = pr.StartPod(ctx)
			c.Assert(err, Not(IsNil))
			c.Assert(errors.Is(err, ErrPodControllerPodAlreadyStarted), Equals, true)
			c.Assert(prp.inCreatePodCli, IsNil)
			c.Assert(prp.inCreatePodOptions, IsNil)
		},
	}

	for l, tc := range cases {
		c.Log(l)

		pcp := &fakePodControllerProcessor{}

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

	simulatedError := errors.New("SimulatedError")

	cases := map[string]func(pcp *fakePodControllerProcessor, pc PodController){
		"Waiting failed because pod not started yet": func(pcp *fakePodControllerProcessor, pc PodController) {
			err := pc.WaitForPodReady(ctx)
			c.Assert(err, Not(IsNil))
			c.Assert(errors.Is(err, ErrPodControllerPodNotStarted), Equals, true)
			c.Assert(pcp.inCreatePodOptions, IsNil)
			c.Assert(pcp.inCreatePodCli, IsNil)
		},
		"Waiting failed due to timeout": func(pcp *fakePodControllerProcessor, pc PodController) {
			pcp.createPodRet = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      podControllerPodName,
					Namespace: podControllerNS,
				},
			}
			err := pc.StartPod(ctx)
			c.Assert(err, IsNil)

			pcp.waitForPodReadyErr = simulatedError
			err = pc.WaitForPodReady(ctx)
			c.Assert(err, Not(IsNil))
			c.Assert(pcp.inWaitForPodReadyPodName, Equals, podControllerPodName)
			c.Assert(pcp.inWaitForPodReadyNamespace, Equals, podControllerNS)
			c.Assert(errors.Is(err, pcp.waitForPodReadyErr), Equals, true)
			c.Assert(err.Error(), Equals, fmt.Sprintf("Pod failed to become ready in time: %s", simulatedError.Error()))
			// Check that POD deletion was also invoked with expected arguments
		},
		"Waiting succeeded": func(pcp *fakePodControllerProcessor, pc PodController) {
			pcp.createPodRet = &corev1.Pod{
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

		pcp := &fakePodControllerProcessor{}

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

	cases := map[string]func(pcp *fakePodControllerProcessor, pc PodController){
		"Pod not started yet": func(pcp *fakePodControllerProcessor, pc PodController) {
			err := pc.StopPod(ctx, 30*time.Second, int64(0))
			c.Assert(err, Not(IsNil))
			c.Assert(errors.Is(err, ErrPodControllerPodNotStarted), Equals, true)
			c.Assert(pcp.inDeletePodPodName, Equals, untouchedStr)
			c.Assert(pcp.inDeletePodNamespace, Equals, untouchedStr)
		},
		"Pod deletion error": func(pcp *fakePodControllerProcessor, pc PodController) {
			pcp.createPodRet = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: podControllerPodName,
				},
			}
			err := pc.StartPod(ctx)
			c.Assert(err, IsNil)

			pcp.deletePodErr = simulatedError
			err = pc.StopPod(ctx, 30*time.Second, int64(0))
			c.Assert(err, Not(IsNil))
			c.Assert(errors.Is(err, simulatedError), Equals, true)
		},
		"Pod successfully deleted": func(pcp *fakePodControllerProcessor, pc PodController) {
			pcp.createPodRet = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: podControllerPodName,
				},
			}
			err := pc.StartPod(ctx)
			c.Assert(err, IsNil)

			err = pc.StopPod(ctx, 30*time.Second, int64(0))
			c.Assert(err, IsNil)
			c.Assert(pcp.inDeletePodPodName, Equals, podControllerPodName)
			c.Assert(pcp.inDeletePodNamespace, Equals, podControllerNS)
			gracePeriodSeconds := int64(0)
			c.Assert(pcp.inDeletePodOptions, DeepEquals, metav1.DeleteOptions{GracePeriodSeconds: &gracePeriodSeconds})
		},
	}

	for l, tc := range cases {
		c.Log(l)

		pcp := &fakePodControllerProcessor{
			inDeletePodNamespace: untouchedStr,
			inDeletePodPodName:   untouchedStr,
			inDeletePodOptions:   metav1.DeleteOptions{},
		}

		pc := NewPodController(cli, &PodOptions{
			Namespace: podControllerNS,
			Name:      podControllerPodName,
		}, WithPodControllerProcessor(pcp))

		tc(pcp, pc)
	}
}

func (s *PodControllerTestSuite) TestPodControllerGetCommandExecutorAndFileWriter(c *C) {
	ctx := context.Background()
	cli := fake.NewSimpleClientset()

	cases := map[string]func(pcp *fakePodControllerProcessor, pc PodController){
		"Pod not started yet": func(_ *fakePodControllerProcessor, pc PodController) {
			pce, err := pc.GetCommandExecutor()
			c.Assert(pce, IsNil)
			c.Assert(err, Not(IsNil))
			c.Assert(errors.Is(err, ErrPodControllerPodNotStarted), Equals, true)

			pfw, err := pc.GetFileWriter()
			c.Assert(pfw, IsNil)
			c.Assert(err, Not(IsNil))
			c.Assert(errors.Is(err, ErrPodControllerPodNotStarted), Equals, true)
		},
		"Pod not ready yet": func(pcp *fakePodControllerProcessor, pc PodController) {
			pcp.createPodRet = &corev1.Pod{
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
		"CommandExecutor successfully returned": func(pcp *fakePodControllerProcessor, pc PodController) {
			pcp.createPodRet = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: podControllerPodName,
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

		pcp := &fakePodControllerProcessor{}

		pc := NewPodController(cli, &PodOptions{
			Namespace: podControllerNS,
			Name:      podControllerPodName,
		}, WithPodControllerProcessor(pcp))

		tc(pcp, pc)
	}
}
