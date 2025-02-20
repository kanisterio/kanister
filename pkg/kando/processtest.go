package kando

import (
	"bytes"
	"context"
	"io"
	"os"
)

func startServer(ctx context.Context, addr string) error {
	rc := newRootCommand()
	rc.SetArgs([]string{"process", "server", "-a", addr})
	rc.SetOut(nil)
	rc.SetErr(nil)
	return rc.ExecuteContext(ctx)
}

func waitSock(ctx context.Context, addr string) error {
	lst, err := os.Lstat(addr)
	for ctx.Err() == nil && (err != nil || lst.Mode()&os.ModeSocket == 0) {
		lst, err = os.Lstat(addr)
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}

type ProcessResult struct {
	Pid   string `json:"pid"`
	State string `json:"state"`
}

func executeCommand(ctx context.Context, stdout, stderr io.Writer, args ...string) error {
	rc := newRootCommand()
	rc.SetErr(stderr)
	rc.SetOut(stdout)
	rc.SetArgs(args)
	return rc.ExecuteContext(ctx)
}

func executeCommandWithReset(ctx context.Context, stdout, stderr *bytes.Buffer, args ...string) error {
	stdout.Reset()
	stderr.Reset()
	return executeCommand(ctx, stdout, stderr, args...)
}
