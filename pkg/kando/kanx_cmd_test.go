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
		err := startServer(ctx, addr)
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

// TestProcessClientOutput check that output command outputs stdout and stderr to their respective FDs.
func (s *KanXCmdSuite) TestProcessClientOutput(c *C) {
	addr := c.MkDir() + "/kanister.sock"
	ctx, can := context.WithCancel(context.Background())
	defer can()
	go func() {
		err := startServer(ctx, addr)
		c.Assert(err, IsNil)
	}()
	err := waitSock(ctx, addr)
	c.Assert(err, IsNil)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "create", "--", "bash", "-c", "echo 'hello world 1' && echo 'hello world 2' 1>&2")
	c.Assert(err, IsNil)
	pr := &ProcessResult{}
	err = json.Unmarshal(stdout.Bytes(), pr)
	c.Assert(err, IsNil)
	c.Assert(stderr.String(), Equals, "")
	// get output
	err = executeCommandWithReset(ctx, stdout, stderr, "process", "client", "-a", addr, "output", pr.Pid)
	c.Assert(err, IsNil)
	c.Assert(stdout.String(), Equals, "hello world 1\n")
	c.Assert(stderr.String(), Equals, "hello world 2\n")
}

// TestProcessClientExecute_RedirectStdout checks that stdout contains JSON process metadata and process output without additional output from logging.
func (s *KanXCmdSuite) TestProcessClientExecute_RedirectStdout(c *C) {
	addr := c.MkDir() + "/kanister.sock"
	ctx, can := context.WithCancel(context.Background())
	defer can()
	go func() {
		err := startServer(ctx, addr)
		c.Assert(err, IsNil)
	}()
	err := waitSock(ctx, addr)
	c.Assert(err, IsNil)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "execute", "echo", "hello world")
	c.Assert(err, IsNil)
	bs := stdout.Bytes()
	pr := &ProcessResult{}
	dc := json.NewDecoder(bytes.NewReader(bs))
	err = dc.Decode(pr)
	c.Assert(err, IsNil)
	c.Assert(dc.More(), Equals, true)
	rest := dc.InputOffset()
	c.Assert(string(bs[rest:]), Equals, "hello world\n")
	c.Assert(stderr.String(), Equals, "")
}

// TestProcessClientExecute_RedirectStderr checks that stderr without additional output from logging.
func (s *KanXCmdSuite) TestProcessClientExecute_RedirectStderr(c *C) {
	addr := c.MkDir() + "/kanister.sock"
	ctx, can := context.WithCancel(context.Background())
	defer can()
	go func() {
		err := startServer(ctx, addr)
		c.Assert(err, IsNil)
	}()
	err := waitSock(ctx, addr)
	c.Assert(err, IsNil)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "execute", "--", "bash", "-c", "echo 'hello world' 1>&2")
	c.Assert(err, IsNil)
	bs := stdout.Bytes()
	pr := &ProcessResult{}
	dc := json.NewDecoder(bytes.NewReader(bs))
	err = dc.Decode(pr)
	c.Assert(err, IsNil)
	c.Assert(stderr.String(), Equals, "hello world\n")
}

// TestProcessClientExecute_Exit1 test that non-zero exit code from the child process is reflected in the kando command.
func (s *KanXCmdSuite) TestProcessClientExecute_Exit1(c *C) {
	exitCode := 0
	addr := c.MkDir() + "/kanister.sock"
	exit = func(n int) {
		exitCode = n
	}
	ctx, can := context.WithCancel(context.Background())
	defer can()
	go func() {
		err := startServer(ctx, addr)
		c.Assert(err, IsNil)
	}()
	err := waitSock(ctx, addr)
	c.Assert(err, IsNil)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "execute", "--", "/bin/bash", "-c", "exit 1")
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "exit status 1")
	c.Assert(exitCode, Equals, 1)
	bs := stdout.Bytes()
	pr := &ProcessResult{}
	dc := json.NewDecoder(bytes.NewReader(bs))
	err = dc.Decode(pr)
	c.Assert(err, IsNil)
	c.Assert(pr.Pid, Not(Equals), "")
	c.Assert(pr.State, Equals, "PROCESS_STATE_RUNNING")
	c.Assert(string(stdout.Bytes()[dc.InputOffset():]), Equals, "\n")
	c.Assert(stderr.String(), Equals, "Error: exit status 1\n")
}

func (s *KanXCmdSuite) TestProcessClientGet(c *C) {
	addr := c.MkDir() + "/kanister.sock"
	ctx, can := context.WithCancel(context.Background())
	defer can()
	go func() {
		err := startServer(ctx, addr)
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
