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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// PodControllerProcessor is an interface wrapping kubernetes API invocation
// it is purposed to be replaced by fake implementation in tests
type PodControllerProcessor interface {
	CreatePod(ctx context.Context, options *PodOptions) (*corev1.Pod, error)
	WaitForPodReady(ctx context.Context, namespace, podName string) error
	WaitForPodCompletion(ctx context.Context, namespace, podName string) error
	DeletePod(ctx context.Context, namespace string, podName string, opts metav1.DeleteOptions) error
}

type podControllerProcessor struct {
	cli kubernetes.Interface
}

func (p *podControllerProcessor) CreatePod(ctx context.Context, options *PodOptions) (*corev1.Pod, error) {
	return CreatePod(ctx, p.cli, options)
}

func (p *podControllerProcessor) WaitForPodReady(ctx context.Context, namespace, podName string) error {
	return WaitForPodReady(ctx, p.cli, namespace, podName)
}

func (p *podControllerProcessor) WaitForPodCompletion(ctx context.Context, namespace, podName string) error {
	return WaitForPodCompletion(ctx, p.cli, namespace, podName)
}

func (p *podControllerProcessor) DeletePod(ctx context.Context, namespace string, podName string, opts metav1.DeleteOptions) error {
	return p.cli.CoreV1().Pods(namespace).Delete(ctx, podName, opts)
}
