package kube

import (
	"context"
	"io"
)

type FakeKubePodCommandExecutor struct {
	ExecErr       error
	inExecCommand []string

	ExecStdout string
	ExecStderr string
}

// Exec
func (fce *FakeKubePodCommandExecutor) Exec(_ context.Context, command []string, stdin io.Reader, stdout, stderr io.Writer) error {
	fce.inExecCommand = make([]string, len(command))
	copy(fce.inExecCommand, command)
	if stdout != nil && len(fce.ExecStdout) > 0 {
		stdout.Write([]byte(fce.ExecStdout)) //nolint: errcheck
	}
	if stderr != nil && len(fce.ExecStderr) > 0 {
		stderr.Write([]byte(fce.ExecStderr)) //nolint: errcheck
	}

	return fce.ExecErr
}
