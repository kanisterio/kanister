package kando

import (
	"bytes"
	"context"
	"io"
	"os"
	"time"
)

func startServer(ctx context.Context, addr string) error {
	rc := newRootCommand()
	rc.SetArgs([]string{"process", "server", "-a", addr})
	rc.SetOut(nil)
	rc.SetErr(nil)
	return rc.ExecuteContext(ctx)
}

// waitSock wait for socket file to appear.  This signifies that the KanX server has started
func waitSock(ctx context.Context, path string) error {
	lst, err := os.Lstat(path)
	// wait until the file exists and ModeSocket is set
	for ctx.Err() == nil && (err != nil || lst.Mode()&os.ModeSocket == 0) {
		// brain-dead sleep - waste time until file state might change
		time.Sleep(100 * time.Millisecond)
		lst, err = os.Lstat(path)
	}
	// return the context error if there is one.
	if ctx.Err() != nil {
		return ctx.Err()
	}
	// otherwise return the error from Lstat
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

func resetBuffers(bufs ...*bytes.Buffer) {
	for _, buf := range bufs {
		buf.Reset()
	}
}
