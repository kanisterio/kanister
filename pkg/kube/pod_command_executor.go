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
	"io"

	"k8s.io/client-go/kubernetes"
)

// PodCommandExecutor provides a way to execute a command within the pod.
// Is intended to be returned by PodController and works with pod controlled by it.
type PodCommandExecutor interface {
	Exec(ctx context.Context, command []string, stdin io.Reader, stdout, stderr io.Writer) error
}

// podCommandExecutor keeps everything required to execute command within a pod
type podCommandExecutor struct {
	cli           kubernetes.Interface
	namespace     string
	podName       string
	containerName string

	pcep PodCommandExecutorProcessor
}

// Exec runs the command and logs stdout and stderr.
// In case of execution error, ExecError produced by ExecWithOptions will be returned
func (p *podCommandExecutor) Exec(ctx context.Context, command []string, stdin io.Reader, stdout, stderr io.Writer) error {
	var (
		opts = ExecOptions{
			Command:       command,
			Namespace:     p.namespace,
			PodName:       p.podName,
			ContainerName: p.containerName,
			Stdin:         stdin,
			Stdout:        stdout,
			Stderr:        stderr,
		}

		cmdDone = make(chan struct{})
		err     error
	)

	go func() {
		err = p.pcep.ExecWithOptions(ctx, opts)
		close(cmdDone)
	}()

	select {
	case <-ctx.Done():
		err = ctx.Err()
	case <-cmdDone:
	}

	return err
}
