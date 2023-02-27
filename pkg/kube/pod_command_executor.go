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

// PodCommandExecutor allows us to execute command within the pod
type PodCommandExecutor interface {
	Exec(ctx context.Context, command []string, stdin io.Reader, stdout, stderr io.Writer) error
}

// podCommandExecutorProcessor aids in unit testing
type podCommandExecutorProcessor interface {
	execWithOptions(cli kubernetes.Interface, opts ExecOptions) (string, string, error)
}

// podCommandExecutor keeps everything required to execute command within a pod
type podCommandExecutor struct {
	cli           kubernetes.Interface
	namespace     string
	podName       string
	containerName string

	pcep podCommandExecutorProcessor
}

// Exec runs the command and logs stdout and stderr.
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
		_, _, err = p.pcep.execWithOptions(p.cli, opts)
		close(cmdDone)
	}()

	select {
	case <-ctx.Done():
		err = ctx.Err()
	case <-cmdDone:
	}

	return err
}

func (p *podCommandExecutor) execWithOptions(cli kubernetes.Interface, opts ExecOptions) (string, string, error) {
	return ExecWithOptions(p.cli, opts)
}
