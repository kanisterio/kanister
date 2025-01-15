package kando

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"testing"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type KanXCmdSuite struct{}

var _ = Suite(&KanXCmdSuite{})

func startServer(ctx context.Context, addr string, stdout, stderr io.Writer) error {
	rc := newRootCommand()
	rc.SetArgs([]string{"process", "server", "-a", addr})
	rc.SetOut(stdout)
	rc.SetErr(stderr)
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

func (s *KanXCmdSuite) TestProcessServer(c *C) {
	addr := c.MkDir() + "/kanister.sock"
	ctx, can := context.WithCancel(context.Background())
	rc := newRootCommand()
	rc.SetArgs([]string{"process", "server", "-a", addr})
	go func() {
		err := rc.ExecuteContext(ctx)
		c.Assert(err, IsNil)
	}()
	err := waitSock(ctx, addr)
	c.Assert(err, IsNil)
	can()
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

func (s *KanXCmdSuite) TestProcessClientCreate(c *C) {
	addr := c.MkDir() + "/kanister.sock"
	ctx, can := context.WithCancel(context.Background())
	defer can()
	go func() {
		err := startServer(ctx, addr, nil, nil)
		c.Assert(err, IsNil)
	}()
	err := waitSock(ctx, addr)
	c.Assert(err, IsNil)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "create", "sleep", "2s")
	c.Assert(err, IsNil)
	pr := &ProcessResult{}
	err = json.Unmarshal(stdout.Bytes(), pr)
	c.Assert(err, IsNil)
	c.Assert(stderr.String(), Equals, "")
	// get output
	err = executeCommandWithReset(ctx, stdout, stderr, "process", "client", "-a", addr, "output", pr.Pid)
	c.Assert(err, IsNil)
	c.Assert(stdout.String(), Equals, "")
	c.Assert(stderr.String(), Equals, "")
}

func (s *KanXCmdSuite) TestProcessClientOutput(c *C) {
	addr := c.MkDir() + "/kanister.sock"
	ctx, can := context.WithCancel(context.Background())
	defer can()
	go func() {
		err := startServer(ctx, addr, nil, nil)
		c.Assert(err, IsNil)
	}()
	err := waitSock(ctx, addr)
	c.Assert(err, IsNil)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "create", "echo", "hello world")
	c.Assert(err, IsNil)
	pr := &ProcessResult{}
	err = json.Unmarshal(stdout.Bytes(), pr)
	c.Assert(err, IsNil)
	c.Assert(stderr.String(), Equals, "")
	// get output
	err = executeCommandWithReset(ctx, stdout, stderr, "process", "client", "-a", addr, "output", pr.Pid)
	c.Assert(err, IsNil)
	c.Assert(stdout.String(), Equals, "hello world\n")
	c.Assert(stderr.String(), Equals, "")
}

func (s *KanXCmdSuite) TestProcessClientGet(c *C) {
	addr := c.MkDir() + "/kanister.sock"
	ctx, can := context.WithCancel(context.Background())
	defer can()
	go func() {
		err := startServer(ctx, addr, nil, nil)
		c.Assert(err, IsNil)
	}()
	err := waitSock(ctx, addr)
	c.Assert(err, IsNil)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "create", "echo", "hello world")
	c.Assert(err, IsNil)
	pr := &ProcessResult{}
	err = json.Unmarshal(stdout.Bytes(), pr)
	c.Assert(err, IsNil)
	c.Assert(pr.Pid, Not(Equals), "")
	c.Assert(pr.State, Equals, "PROCESS_STATE_RUNNING")
	c.Assert(stderr.String(), Equals, "")
	// get output
	err = executeCommandWithReset(ctx, stdout, stderr, "process", "client", "-a", addr, "output", pr.Pid)
	c.Assert(err, IsNil)
	c.Assert(stdout.String(), Equals, "hello world\n")
	c.Assert(stderr.String(), Equals, "")
	err = executeCommandWithReset(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "get", pr.Pid)
	c.Assert(err, IsNil)
	pr = &ProcessResult{}
	err = json.Unmarshal(stdout.Bytes(), pr)
	c.Assert(err, IsNil)
	c.Assert(pr.Pid, Not(Equals), "")
	c.Assert(pr.State, Equals, "PROCESS_STATE_SUCCEEDED")
	c.Assert(stderr.String(), Equals, "")
}
