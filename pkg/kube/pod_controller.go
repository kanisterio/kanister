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
	StartPod(ctx context.Context, stopTimeout time.Duration) error
	WaitForPodReady(ctx context.Context) error
	StopPod(ctx context.Context) error
	GetCommandExecutor() (PodCommandExecutor, error)
	GetFileWriter() (PodFileWriter, error)
}

// podControllerProcessor aids in unit testing
type podControllerProcessor interface {
	createPod(ctx context.Context, cli kubernetes.Interface, options *PodOptions) (*corev1.Pod, error)
	waitForPodReady(ctx context.Context, podName string) error
	deletePod(ctx context.Context, namespace string, podName string, opts metav1.DeleteOptions) error
}

// podController specifies Kubernetes Client and PodOptions needed for creating Pod
type podController struct {
	cli        kubernetes.Interface
	podOptions *PodOptions

	pod         *corev1.Pod
	podReady    bool
	podName     string
	stopTimeout time.Duration

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
// stopTimeout is also stored and will be used when StopPod will be called
func (p *podController) StartPod(ctx context.Context, stopTimeout time.Duration) error {
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
	p.stopTimeout = stopTimeout

	return nil
}

// WaitForPod waits for POD readiness.
func (p *podController) WaitForPodReady(ctx context.Context) error {
	if p.podName == "" {
		return errors.Wrap(ErrPodControllerPodNotStarted, "Pod failed to become ready in time")
	}

	if err := p.pcp.waitForPodReady(ctx, p.podName); err != nil {
		log.WithError(err).Print("Pod failed to become ready in time", field.M{"PodName": p.podName, "Namespace": p.podOptions.Namespace})
		_ = p.StopPod(ctx) // best-effort
		return errors.Wrap(err, "Pod failed to become ready in time")
	}

	p.podReady = true

	return nil
}

// StopPod stops the pod which was previously started, otherwise it will return ErrPodControllerPodNotStarted error.
// stopTimeout passed to Start will be used
func (p *podController) StopPod(ctx context.Context) error {
	if p.podName == "" {
		return ErrPodControllerPodNotStarted
	}

	if p.stopTimeout != PodControllerInfiniteStopTime {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.stopTimeout)
		defer cancel()
	}

	gracePeriodSeconds := int64(0) // force immediate deletion

	if err := p.pcp.deletePod(ctx, p.podOptions.Namespace, p.podName, metav1.DeleteOptions{GracePeriodSeconds: &gracePeriodSeconds}); err != nil {
		log.WithError(err).Print("Failed to delete pod", field.M{"PodName": p.podName, "Namespace": p.podOptions.Namespace})
		return err
	}

	p.podReady = false
	p.podName = ""
	p.pod = nil

	return nil
}

// This is wrapped for unit testing.
func (p *podController) createPod(ctx context.Context, cli kubernetes.Interface, options *PodOptions) (*corev1.Pod, error) {
	return CreatePod(ctx, cli, options)
}

// This is wrapped for unit testing.
func (p *podController) waitForPodReady(ctx context.Context, podName string) error {
	return WaitForPodReady(ctx, p.cli, p.podOptions.Namespace, podName)
}

// This is wrapped for unit testing.
func (p *podController) deletePod(ctx context.Context, namespace string, podName string, opts metav1.DeleteOptions) error {
	return p.cli.CoreV1().Pods(namespace).Delete(ctx, podName, opts)
}
