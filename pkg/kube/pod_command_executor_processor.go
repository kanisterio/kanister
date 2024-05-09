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

	"k8s.io/client-go/kubernetes"
)

// PodCommandExecutorProcessor is an interface wrapping kubernetes API invocation
// it is purposed to be replaced by fake implementation in tests
type PodCommandExecutorProcessor interface {
	ExecWithOptions(ctx context.Context, opts ExecOptions) error
}

type podCommandExecutorProcessor struct {
	cli kubernetes.Interface
}

// ExecWithOptions executes a command in the specified pod and container,
// returning stdout, stderr and error.
func (p *podCommandExecutorProcessor) ExecWithOptions(ctx context.Context, opts ExecOptions) error {
	return ExecWithOptions(ctx, p.cli, opts)
}
