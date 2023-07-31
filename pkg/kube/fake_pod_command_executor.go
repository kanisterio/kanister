package kube

import (
	"context"
	"io"
)

// // FakePodCommandExecutor allows us to execute command within the pod
// type FakePodCommandExecutor struct {
// 	InExecWithOptionsCli  kubernetes.Interface
// 	InExecWithOptionsOpts *ExecOptions
// 	ExecWithOptionsStdout string
// 	ExecWithOptionsStderr string
// 	ExecWithOptionsRet1   string
// 	ExecWithOptionsRet2   string
// 	ExecWithOptionsErr    error
// }

type FakeKubePodCommandExecutor struct {
	ExecErr       error
	inExecCommand []string

	ExecStdout string
	ExecStderr string
}

// Exec runs the command and logs stdout and stderr.
// func (p *FakeKubePodCommandExecutor) Exec(ctx context.Context, command []string, stdin io.Reader, stdout, stderr io.Writer) error {

// 	return errors.New("")
// }

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
