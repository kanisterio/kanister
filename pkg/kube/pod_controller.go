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

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
)

var (
	ErrPodControllerNotInitialized    = errors.New("pod has not been initialized")
	ErrPodControllerPodAlreadyStarted = errors.New("pod has already been started")
	ErrPodControllerPodNotReady       = errors.New("pod is not yet ready")
	ErrPodControllerPodNotStarted     = errors.New("pod is not yet started")
	PodControllerDefaultStopTime      = 30 * time.Second
	PodControllerInfiniteStopTime     = 0 * time.Second
)

// PodController specifies interface needed for starting, stopping pod and operations with it
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

// podControllerProcessor aids in unit testing
type podControllerProcessor interface {
	createPod(ctx context.Context, cli kubernetes.Interface, options *PodOptions) (*corev1.Pod, error)
	waitForPodReady(ctx context.Context, namespace, podName string) error
	waitForPodCompletion(ctx context.Context, namespace, podName string) error
	deletePod(ctx context.Context, namespace string, podName string, opts metav1.DeleteOptions) error
}

// podController specifies Kubernetes Client and PodOptions needed for creating Pod
type podController struct {
	cli        kubernetes.Interface
	podOptions *PodOptions

	pod      *corev1.Pod
	podReady bool
	podName  string

	pcp podControllerProcessor
}

type PodControllerOption func(p *podController)

// WithPodControllerProcessor provides mechanism for passing fake podControllerProcessor for testing purposes.
func WithPodControllerProcessor(processor podControllerProcessor) PodControllerOption {
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

	r.pcp = r

	for _, opt := range opts {
		opt(r)
	}

	return r
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
		return errors.Wrap(ErrPodControllerPodAlreadyStarted, "Failed to create pod")
	}

	if p.cli == nil || p.podOptions == nil {
		return errors.Wrap(ErrPodControllerNotInitialized, "Failed to create pod")
	}

	pod, err := p.pcp.createPod(ctx, p.cli, p.podOptions)
	if err != nil {
		log.WithError(err).Print("Failed to create pod", field.M{"PodName": p.podOptions.Name, "Namespace": p.podOptions.Namespace})
		return errors.Wrap(err, "Failed to create pod")
	}

	p.pod = pod
	p.podName = pod.Name

	return nil
}

// WaitForPod waits for pod readiness.
func (p *podController) WaitForPodReady(ctx context.Context) error {
	if p.podName == "" {
		return ErrPodControllerPodNotStarted
	}

	if err := p.pcp.waitForPodReady(ctx, p.pod.Namespace, p.pod.Name); err != nil {
		log.WithError(err).Print("Pod failed to become ready in time", field.M{"PodName": p.podName, "Namespace": p.podOptions.Namespace})
		return errors.Wrap(err, "Pod failed to become ready in time")
	}

	p.podReady = true

	return nil
}

// WaitForPodCompletion waits for a pod to reach a terminal state.
func (p *podController) WaitForPodCompletion(ctx context.Context) error {
	if p.podName == "" {
		return ErrPodControllerPodNotStarted
	}

	if !p.podReady {
		return ErrPodControllerPodNotReady
	}

	if err := p.pcp.waitForPodCompletion(ctx, p.pod.Namespace, p.pod.Name); err != nil {
		log.WithError(err).Print("Pod failed to complete in time", field.M{"PodName": p.podName, "Namespace": p.podOptions.Namespace})
		return errors.Wrap(err, "Pod failed to complete in time")
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

	if err := p.pcp.deletePod(ctx, p.podOptions.Namespace, p.podName, metav1.DeleteOptions{GracePeriodSeconds: &gracePeriodSeconds}); err != nil {
		log.WithError(err).Print("Failed to delete pod", field.M{"PodName": p.podName, "Namespace": p.podOptions.Namespace})
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

	return StreamPodLogs(ctx, p.cli, p.pod.Namespace, p.pod.Name, p.pod.Spec.Containers[0].Name)
}

func (p *podController) GetCommandExecutor() (PodCommandExecutor, error) {
	if p.podName == "" {
		return nil, ErrPodControllerPodNotStarted
	}

	if !p.podReady {
		return nil, ErrPodControllerPodNotReady
	}

	pce := &podCommandExecutor{
		cli:           p.cli,
		namespace:     p.podOptions.Namespace,
		podName:       p.podName,
		containerName: p.podOptions.ContainerName,
	}

	pce.pcep = pce

	return pce, nil
}

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
		containerName: p.podOptions.ContainerName,
	}

	pfw.fileWriterProcessor = pfw

	return pfw, nil
}

// This is wrapped for unit testing.
func (p *podController) createPod(ctx context.Context, cli kubernetes.Interface, options *PodOptions) (*corev1.Pod, error) {
	return CreatePod(ctx, cli, options)
}

// This is wrapped for unit testing.
func (p *podController) waitForPodReady(ctx context.Context, namespace, podName string) error {
	return WaitForPodReady(ctx, p.cli, namespace, podName)
}

// This is wrapped for unit testing
func (p *podController) waitForPodCompletion(ctx context.Context, namespace, podName string) error {
	return WaitForPodCompletion(ctx, p.cli, namespace, podName)
}

// This is wrapped for unit testing.
func (p *podController) deletePod(ctx context.Context, namespace string, podName string, opts metav1.DeleteOptions) error {
	return p.cli.CoreV1().Pods(namespace).Delete(ctx, podName, opts)
}
