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

// ExecError is an error returned by PodCommandExecutor.Exec
// It contains not only error happened during an execution, but also keeps tails of stdout/stderr streams.
// These tails could be used by the invoker to construct more precise error.
type ExecError struct {
	error
	stdout LogTail
	stderr LogTail
}

// NewExecError creates an instance of ExecError
func NewExecError(err error, stdout, stderr LogTail) *ExecError {
	return &ExecError{
		error:  err,
		stdout: stdout,
		stderr: stderr,
	}
}

func (e *ExecError) Unwrap() error {
	return e.error
}

func (e *ExecError) Stdout() string {
	return e.stdout.ToString()
}

func (e *ExecError) Stderr() string {
	return e.stderr.ToString()
}

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
// In case of execution error, ExecError will be returned
func (p *podCommandExecutor) Exec(ctx context.Context, command []string, stdin io.Reader, stdout, stderr io.Writer) error {
	var (
		stderrTail = NewLogTail(logTailDefaultLength)
		stdoutTail = NewLogTail(logTailDefaultLength)
		opts       = ExecOptions{
			Command:       command,
			Namespace:     p.namespace,
			PodName:       p.podName,
			ContainerName: p.containerName,
			Stdin:         stdin,
			Stdout:        stdoutTail,
			Stderr:        stderrTail,
		}

		cmdDone = make(chan struct{})
		err     error
	)

	if stdout != nil {
		opts.Stdout = io.MultiWriter(stdout, stdoutTail)
	}
	if stderr != nil {
		opts.Stderr = io.MultiWriter(stderr, stderrTail)
	}

	go func() {
		_, _, err = p.pcep.ExecWithOptions(opts)
		close(cmdDone)
	}()

	select {
	case <-ctx.Done():
		err = ctx.Err()
	case <-cmdDone:
		if err != nil {
			err = &ExecError{
				error:  err,
				stdout: stdoutTail,
				stderr: stderrTail,
			}
		}
	}

	return err
}
