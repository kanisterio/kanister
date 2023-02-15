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

	corev1 "k8s.io/api/core/v1"
)

type PodController interface {
	PodName() string
	Pod() *corev1.Pod
	StartPod(ctx context.Context, stopTimeout time.Duration) error
	WaitForPodReady(ctx context.Context) error
	StopPod(ctx context.Context) error
	GetCommandExecutor() (PodCommandExecutor, error)
	GetFileWriter() (PodFileWriter, error)
}
