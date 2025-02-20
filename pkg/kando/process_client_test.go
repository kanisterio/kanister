package kando

import (
	"bytes"
	"context"
	"encoding/json"

	. "gopkg.in/check.v1"
)

type KanXCmdProcessClientSuite struct{}

var _ = Suite(&KanXCmdProcessClientSuite{})

func (s *KanXCmdProcessClientSuite) TestProcessClientCreate(c *C) {
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

func (s *KanXCmdProcessClientSuite) TestProcessClientList_0(c *C) {
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
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "list")
	c.Assert(err, IsNil)
	pr := &ProcessResult{}
	dcd := json.NewDecoder(stdout)
	n := 0
	for dcd.More() {
		err = dcd.Decode(pr)
		n++
	}
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 0)
	c.Assert(stderr.String(), Equals, "")
}

func (s *KanXCmdProcessClientSuite) TestProcessClientList_1(c *C) {
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
	stdout.Reset()
	stderr.Reset()
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "create", "sleep", "2s")
	c.Assert(err, IsNil)
	stdout.Reset()
	stderr.Reset()
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "list")
	c.Assert(err, IsNil)
	pr := &ProcessResult{}
	dcd := json.NewDecoder(stdout)
	n := 0
	for dcd.More() {
		err = dcd.Decode(pr)
		n++
	}
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 2)
	c.Assert(stderr.String(), Equals, "")
}

func (s *KanXCmdProcessClientSuite) TestProcessClientSignal(c *C) {
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
	c.Assert(pr.State, Equals, "PROCESS_STATE_RUNNING")
	c.Assert(stderr.String(), Equals, "")
	stdout.Reset()
	stderr.Reset()
	err = executeCommandWithReset(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "signal", pr.Pid, "2")
	c.Assert(err, IsNil)
	err = json.Unmarshal(stdout.Bytes(), pr)
	c.Assert(err, IsNil)
	c.Assert(pr.State, Equals, "PROCESS_STATE_RUNNING")
	c.Assert(stderr.String(), Equals, "")
	stdout.Reset()
	stderr.Reset()
	err = executeCommandWithReset(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "output", pr.Pid)
	c.Assert(err, IsNil)
	c.Assert(stdout.String(), Equals, "")
	c.Assert(stderr.String(), Equals, "")
	stdout.Reset()
	stderr.Reset()
	err = executeCommandWithReset(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "get", pr.Pid)
	c.Assert(err, IsNil)
	err = json.Unmarshal(stdout.Bytes(), pr)
	c.Assert(err, IsNil)
	c.Assert(pr.State, Equals, "PROCESS_STATE_FAILED")
	c.Assert(stderr.String(), Equals, "")
}

// TestProcessClientOutput check that output command outputs stdout and stderr to their respective FDs.
func (s *KanXCmdProcessClientSuite) TestProcessClientOutput(c *C) {
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
func (s *KanXCmdProcessClientSuite) TestProcessClientExecute_RedirectStdout(c *C) {
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
func (s *KanXCmdProcessClientSuite) TestProcessClientExecute_RedirectStderr(c *C) {
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
func (s *KanXCmdProcessClientSuite) TestProcessClientExecute_Exit1(c *C) {
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
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--exit-proxy", "--as-json", "-a", addr, "execute", "--", "/bin/bash", "-c", "exit 1")
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

func (s *KanXCmdProcessClientSuite) TestProcessClientGet_0(c *C) {
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

func (s *KanXCmdProcessClientSuite) TestProcessClientGet_1(c *C) {
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
	err = executeCommandWithReset(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "get", "555555555")
	c.Assert(err, NotNil)
	c.Assert(stderr.String(), Matches, ".*Process not found.*\n")
}
