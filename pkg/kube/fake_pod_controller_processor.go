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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FakePodControllerProcessor implements PodControllerProcessor
type FakePodControllerProcessor struct {
	InWaitForPodReadyNamespace string
	InWaitForPodReadyPodName   string
	WaitForPodReadyErr         error

	InWaitForPodCompletionNamespace string
	InWaitForPodCompletionPodName   string
	WaitForPodCompletionErr         error

	InDeletePodNamespace string
	InDeletePodPodName   string
	InDeletePodOptions   metav1.DeleteOptions
	DeletePodErr         error

	InCreatePodOptions *PodOptions
	CreatePodRet       *corev1.Pod
	CreatePodErr       error
}

func (f *FakePodControllerProcessor) CreatePod(_ context.Context, options *PodOptions) (*corev1.Pod, error) {
	f.InCreatePodOptions = options
	return f.CreatePodRet, f.CreatePodErr
}

func (f *FakePodControllerProcessor) WaitForPodCompletion(_ context.Context, namespace, podName string) error {
	f.InWaitForPodCompletionNamespace = namespace
	f.InWaitForPodCompletionPodName = podName
	return f.WaitForPodCompletionErr
}

func (f *FakePodControllerProcessor) WaitForPodReady(_ context.Context, namespace, podName string) error {
	f.InWaitForPodReadyPodName = podName
	f.InWaitForPodReadyNamespace = namespace
	return f.WaitForPodReadyErr
}

func (f *FakePodControllerProcessor) DeletePod(_ context.Context, namespace string, podName string, opts metav1.DeleteOptions) error {
	f.InDeletePodNamespace = namespace
	f.InDeletePodPodName = podName
	f.InDeletePodOptions = opts

	return f.DeletePodErr
}
