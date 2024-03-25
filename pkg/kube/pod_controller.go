// Copyright 2023 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kube

import (
	"context"
	"io"
	"time"

	"github.com/kanisterio/errkit"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
)

var (
	ErrPodControllerNotInitialized    = errkit.NewSentinelErr("pod has not been initialized")
	ErrPodControllerPodAlreadyStarted = errkit.NewSentinelErr("pod has already been started")
	ErrPodControllerPodNotReady       = errkit.NewSentinelErr("pod is not yet ready")
	ErrPodControllerPodNotStarted     = errkit.NewSentinelErr("pod is not yet started")
	PodControllerDefaultStopTime      = 30 * time.Second
	PodControllerInfiniteStopTime     = 0 * time.Second
)

// PodController specifies interface needed for starting, stopping pod and operations with it
//
// The purpose of this interface is to provide single mechanism of pod manipulation, reduce number of parameters which
// caller needs to pass (since we keep pod related things internally) and eliminate human errors.
type PodController interface {
	PodName() string
	Pod() *corev1.Pod
	StartPod(ctx context.Context) error
	WaitForPodReady(ctx context.Context) error
	WaitForPodCompletion(ctx context.Context) error
	StopPod(ctx context.Context, timeout time.Duration, gracePeriodSeconds int64) error

	StreamPodLogs(ctx context.Context) (io.ReadCloser, error)

	GetCommandExecutor() (PodCommandExecutor, error)
	GetFileWriter() (PodFileWriter, error)
}

// podController keeps Kubernetes Client and PodOptions needed for creating a Pod.
// It implements the PodControllerProcessor interface.
// All communication with kubernetes API are done via PodControllerProcessor interface, which could be overridden for testing purposes.
type podController struct {
	cli        kubernetes.Interface
	podOptions *PodOptions

	pod      *corev1.Pod
	podReady bool
	podName  string

	pcp PodControllerProcessor
}

type PodControllerOption func(p *podController)

// WithPodControllerProcessor provides mechanism for passing fake implementation of PodControllerProcessor for testing purposes.
func WithPodControllerProcessor(processor PodControllerProcessor) PodControllerOption {
	return func(p *podController) {
		p.pcp = processor
	}
}

// NewPodController returns a new PodController given Kubernetes Client and PodOptions
func NewPodController(cli kubernetes.Interface, options *PodOptions, opts ...PodControllerOption) PodController {
	r := &podController{
		cli:        cli,
		podOptions: options,
	}

	for _, opt := range opts {
		opt(r)
	}

	// If pod controller processor has not been set by PodControllerOption, we create default implementation here
	if r.pcp == nil {
		r.pcp = &podControllerProcessor{
			cli: cli,
		}
	}

	return r
}

// NewPodControllerForExistingPod returns a new PodController for the given
// running pod.
// Invocation of StartPod of returned PodController instance will fail, since
// the pod is expected to be running already.
// Note:
// If the pod is not in the ready state, it will wait for up to
// KANISTER_POD_READY_WAIT_TIMEOUT (15 minutes by default) until the pod becomes ready.
func NewPodControllerForExistingPod(cli kubernetes.Interface, pod *corev1.Pod) (PodController, error) {
	err := WaitForPodReady(context.Background(), cli, pod.Namespace, pod.Name)
	if err != nil {
		return nil, err
	}

	pc := &podController{
		cli: cli,
		pcp: &podControllerProcessor{
			cli: cli,
		},
		pod:     pod,
		podName: pod.Name,
	}

	options := &PodOptions{
		Name:          pod.Name,
		Namespace:     pod.Namespace,
		ContainerName: pod.Spec.Containers[0].Name,
	}
	pc.podOptions = options
	pc.podReady = true

	return pc, nil
}

func (p *podController) PodName() string {
	return p.podName
}

func (p *podController) Pod() *corev1.Pod {
	return p.pod
}

// StartPod creates pod and in case of success, it stores pod name for further use.
func (p *podController) StartPod(ctx context.Context) error {
	if p.podName != "" {
		return errkit.Wrap(ErrPodControllerPodAlreadyStarted, "Failed to create pod")
	}

	if p.cli == nil || p.podOptions == nil {
		return errkit.Wrap(ErrPodControllerNotInitialized, "Failed to create pod")
	}

	pod, err := p.pcp.CreatePod(ctx, p.podOptions)
	if err != nil {
		log.WithError(err).Print("Failed to create pod", field.M{"PodName": p.podOptions.Name, "Namespace": p.podOptions.Namespace})
		return errkit.Wrap(err, "Failed to create pod")
	}

	p.pod = pod
	p.podName = pod.Name

	return nil
}

// WaitForPodReady waits for pod readiness (actually it waits until pod exit the pending state)
// Requires pod to be started otherwise will immediately fail.
func (p *podController) WaitForPodReady(ctx context.Context) error {
	if p.podName == "" {
		return ErrPodControllerPodNotStarted
	}

	if err := p.pcp.WaitForPodReady(ctx, p.pod.Namespace, p.pod.Name); err != nil {
		log.WithError(err).Print("Pod failed to become ready in time", field.M{"PodName": p.podName, "Namespace": p.podOptions.Namespace})
		return errkit.Wrap(err, "Pod failed to become ready in time")
	}

	p.podReady = true

	return nil
}

// WaitForPodCompletion waits for a pod to reach a terminal (either succeeded or failed) state.
func (p *podController) WaitForPodCompletion(ctx context.Context) error {
	if p.podName == "" {
		return ErrPodControllerPodNotStarted
	}

	if !p.podReady {
		return ErrPodControllerPodNotReady
	}

	if err := p.pcp.WaitForPodCompletion(ctx, p.pod.Namespace, p.pod.Name); err != nil {
		log.WithError(err).Print("Pod failed to complete in time", field.M{"PodName": p.podName, "Namespace": p.podOptions.Namespace})
		return errkit.Wrap(err, "Pod failed to complete in time")
	}

	p.podReady = false

	return nil
}

// StopPod stops the pod which was previously started, otherwise it will return ErrPodControllerPodNotStarted error.
// stopTimeout is used to limit execution time
// gracePeriodSeconds is used to specify pod deletion grace period. If set to zero, pod should be deleted immediately
func (p *podController) StopPod(ctx context.Context, stopTimeout time.Duration, gracePeriodSeconds int64) error {
	if p.podName == "" {
		return ErrPodControllerPodNotStarted
	}

	if stopTimeout != PodControllerInfiniteStopTime {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, stopTimeout)
		defer cancel()
	}

	if err := p.pcp.DeletePod(ctx, p.pod.Namespace, p.podName, metav1.DeleteOptions{GracePeriodSeconds: &gracePeriodSeconds}); err != nil {
		log.WithError(err).Print("Failed to delete pod", field.M{"PodName": p.podName, "Namespace": p.pod.Namespace})
		return err
	}

	p.podReady = false
	p.podName = ""
	p.pod = nil

	return nil
}

func (p *podController) StreamPodLogs(ctx context.Context) (io.ReadCloser, error) {
	if p.podName == "" {
		return nil, ErrPodControllerPodNotStarted
	}

	return StreamPodLogs(ctx, p.cli, p.pod.Namespace, p.pod.Name, ContainerNameFromPodOptsOrDefault(p.podOptions))
}

// GetCommandExecutor returns PodCommandExecutor instance which is configured to execute commands within pod controlled
// by this controller.
// If pod is not created or not ready yet, it will fail with an appropriate error.
// Container will be decided based on the result of getContainerName function
func (p *podController) GetCommandExecutor() (PodCommandExecutor, error) {
	if p.podName == "" {
		return nil, ErrPodControllerPodNotStarted
	}

	if !p.podReady {
		return nil, ErrPodControllerPodNotReady
	}

	pce := &podCommandExecutor{
		cli:           p.cli,
		namespace:     p.pod.Namespace,
		podName:       p.podName,
		containerName: ContainerNameFromPodOptsOrDefault(p.podOptions),
	}

	pce.pcep = &podCommandExecutorProcessor{
		cli: p.cli,
	}

	return pce, nil
}

// GetFileWriter returns PodFileWriter instance which is configured to write file to pod controlled by this controller.
// If pod is not created or not ready yet, it will fail with an appropriate error.
// Container will be decided based on the result of getContainerName function
func (p *podController) GetFileWriter() (PodFileWriter, error) {
	if p.podName == "" {
		return nil, ErrPodControllerPodNotStarted
	}

	if !p.podReady {
		return nil, ErrPodControllerPodNotReady
	}

	pfw := &podFileWriter{
		cli:           p.cli,
		namespace:     p.podOptions.Namespace,
		podName:       p.podName,
		containerName: ContainerNameFromPodOptsOrDefault(p.podOptions),
	}

	pfw.fileWriterProcessor = &podFileWriterProcessor{
		cli: p.cli,
	}

	return pfw, nil
}
