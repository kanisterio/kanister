package kando

import (
	"bytes"
	"context"
	"encoding/json"

	check "gopkg.in/check.v1"
)

type KanXCmdProcessClientSuite struct{}

var _ = check.Suite(&KanXCmdProcessClientSuite{})

func (s *KanXCmdProcessClientSuite) TestProcessClientCreate(c *check.C) {
	addr := c.MkDir() + "/kanister.sock"
	ctx, can := context.WithCancel(context.Background())
	defer can()
	go func() {
		err := startServer(ctx, addr)
		c.Assert(err, check.IsNil)
	}()
	err := waitSock(ctx, addr)
	c.Assert(err, check.IsNil)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "create", "sleep", "2s")
	c.Assert(err, check.IsNil)
	pr := &ProcessResult{}
	err = json.Unmarshal(stdout.Bytes(), pr)
	c.Assert(err, check.IsNil)
	c.Assert(stderr.String(), check.Equals, "")
	// get output
	resetBuffers(stdout, stderr)
	err = executeCommand(ctx, stdout, stderr, "process", "client", "-a", addr, "output", pr.Pid)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "")
	c.Assert(stderr.String(), check.Equals, "")
}

func (s *KanXCmdProcessClientSuite) TestProcessClientList_0(c *check.C) {
	addr := c.MkDir() + "/kanister.sock"
	ctx, can := context.WithCancel(context.Background())
	defer can()
	go func() {
		err := startServer(ctx, addr)
		c.Assert(err, check.IsNil)
	}()
	err := waitSock(ctx, addr)
	c.Assert(err, check.IsNil)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "list")
	c.Assert(err, check.IsNil)
	pr := &ProcessResult{}
	dcd := json.NewDecoder(stdout)
	n := 0
	for dcd.More() {
		err = dcd.Decode(pr)
		n++
	}
	c.Assert(err, check.IsNil)
	c.Assert(n, check.Equals, 0)
	c.Assert(stderr.String(), check.Equals, "")
}

func (s *KanXCmdProcessClientSuite) TestProcessClientList_1(c *check.C) {
	addr := c.MkDir() + "/kanister.sock"
	ctx, can := context.WithCancel(context.Background())
	defer can()
	go func() {
		err := startServer(ctx, addr)
		c.Assert(err, check.IsNil)
	}()
	err := waitSock(ctx, addr)
	c.Assert(err, check.IsNil)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "create", "sleep", "2s")
	c.Assert(err, check.IsNil)
	resetBuffers(stdout, stderr)
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "create", "sleep", "2s")
	c.Assert(err, check.IsNil)
	resetBuffers(stdout, stderr)
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "list")
	c.Assert(err, check.IsNil)
	pr := &ProcessResult{}
	dcd := json.NewDecoder(stdout)
	n := 0
	for dcd.More() {
		err = dcd.Decode(pr)
		n++
	}
	c.Assert(err, check.IsNil)
	c.Assert(n, check.Equals, 2)
	c.Assert(stderr.String(), check.Equals, "")
}

func (s *KanXCmdProcessClientSuite) TestProcessClientSignal(c *check.C) {
	addr := c.MkDir() + "/kanister.sock"
	ctx, can := context.WithCancel(context.Background())
	defer can()
	go func() {
		err := startServer(ctx, addr)
		c.Assert(err, check.IsNil)
	}()
	err := waitSock(ctx, addr)
	c.Assert(err, check.IsNil)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "create", "sleep", "2s")
	c.Assert(err, check.IsNil)
	pr := &ProcessResult{}
	err = json.Unmarshal(stdout.Bytes(), pr)
	c.Assert(err, check.IsNil)
	c.Assert(pr.State, check.Equals, "PROCESS_STATE_RUNNING")
	c.Assert(stderr.String(), check.Equals, "")
	resetBuffers(stdout, stderr)
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "signal", pr.Pid, "2")
	c.Assert(err, check.IsNil)
	err = json.Unmarshal(stdout.Bytes(), pr)
	c.Assert(err, check.IsNil)
	c.Assert(pr.State, check.Equals, "PROCESS_STATE_RUNNING")
	c.Assert(stderr.String(), check.Equals, "")
	resetBuffers(stdout, stderr)
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "output", pr.Pid)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "")
	c.Assert(stderr.String(), check.Equals, "")
	resetBuffers(stdout, stderr)
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "get", pr.Pid)
	c.Assert(err, check.IsNil)
	err = json.Unmarshal(stdout.Bytes(), pr)
	c.Assert(err, check.IsNil)
	c.Assert(pr.State, check.Equals, "PROCESS_STATE_FAILED")
	c.Assert(stderr.String(), check.Equals, "")
}

// TestProcessClientOutput check that output command outputs stdout and stderr to their respective FDs.
func (s *KanXCmdProcessClientSuite) TestProcessClientOutput(c *check.C) {
	addr := c.MkDir() + "/kanister.sock"
	ctx, can := context.WithCancel(context.Background())
	defer can()
	go func() {
		err := startServer(ctx, addr)
		c.Assert(err, check.IsNil)
	}()
	err := waitSock(ctx, addr)
	c.Assert(err, check.IsNil)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "create", "--", "bash", "-c", "echo 'hello world 1' && echo 'hello world 2' 1>&2")
	c.Assert(err, check.IsNil)
	pr := &ProcessResult{}
	err = json.Unmarshal(stdout.Bytes(), pr)
	c.Assert(err, check.IsNil)
	c.Assert(stderr.String(), check.Equals, "")
	// get output
	resetBuffers(stdout, stderr)
	err = executeCommand(ctx, stdout, stderr, "process", "client", "-a", addr, "output", pr.Pid)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "hello world 1\n")
	c.Assert(stderr.String(), check.Equals, "hello world 2\n")
}

// TestProcessClientExecute_RedirectStdout checks that stdout contains JSON process metadata and process output without additional output from logging.
func (s *KanXCmdProcessClientSuite) TestProcessClientExecute_RedirectStdout(c *check.C) {
	addr := c.MkDir() + "/kanister.sock"
	ctx, can := context.WithCancel(context.Background())
	defer can()
	go func() {
		err := startServer(ctx, addr)
		c.Assert(err, check.IsNil)
	}()
	err := waitSock(ctx, addr)
	c.Assert(err, check.IsNil)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "execute", "echo", "hello world")
	c.Assert(err, check.IsNil)
	bs := stdout.Bytes()
	pr := &ProcessResult{}
	dc := json.NewDecoder(bytes.NewReader(bs))
	err = dc.Decode(pr)
	c.Assert(err, check.IsNil)
	c.Assert(dc.More(), check.Equals, true)
	rest := dc.InputOffset()
	c.Assert(string(bs[rest:]), check.Equals, "hello world\n")
	c.Assert(stderr.String(), check.Equals, "")
}

// TestProcessClientExecute_RedirectStderr checks that stderr without additional output from logging.
func (s *KanXCmdProcessClientSuite) TestProcessClientExecute_RedirectStderr(c *check.C) {
	addr := c.MkDir() + "/kanister.sock"
	ctx, can := context.WithCancel(context.Background())
	defer can()
	go func() {
		err := startServer(ctx, addr)
		c.Assert(err, check.IsNil)
	}()
	err := waitSock(ctx, addr)
	c.Assert(err, check.IsNil)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "execute", "--", "bash", "-c", "echo 'hello world' 1>&2")
	c.Assert(err, check.IsNil)
	bs := stdout.Bytes()
	pr := &ProcessResult{}
	dc := json.NewDecoder(bytes.NewReader(bs))
	err = dc.Decode(pr)
	c.Assert(err, check.IsNil)
	c.Assert(stderr.String(), check.Equals, "hello world\n")
}

func (s *KanXCmdProcessClientSuite) TestProcessClientGet_0(c *check.C) {
	addr := c.MkDir() + "/kanister.sock"
	ctx, can := context.WithCancel(context.Background())
	defer can()
	go func() {
		err := startServer(ctx, addr)
		c.Assert(err, check.IsNil)
	}()
	err := waitSock(ctx, addr)
	c.Assert(err, check.IsNil)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "create", "echo", "hello world")
	c.Assert(err, check.IsNil)
	pr := &ProcessResult{}
	err = json.Unmarshal(stdout.Bytes(), pr)
	c.Assert(err, check.IsNil)
	c.Assert(pr.Pid, Not(Equals), "")
	c.Assert(pr.State, check.Equals, "PROCESS_STATE_RUNNING")
	c.Assert(stderr.String(), check.Equals, "")
	// get output
	resetBuffers(stdout, stderr)
	err = executeCommand(ctx, stdout, stderr, "process", "client", "-a", addr, "output", pr.Pid)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "hello world\n")
	c.Assert(stderr.String(), check.Equals, "")
	resetBuffers(stdout, stderr)
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "get", pr.Pid)
	c.Assert(err, check.IsNil)
	pr = &ProcessResult{}
	err = json.Unmarshal(stdout.Bytes(), pr)
	c.Assert(err, check.IsNil)
	c.Assert(pr.Pid, Not(Equals), "")
	c.Assert(pr.State, check.Equals, "PROCESS_STATE_SUCCEEDED")
	c.Assert(stderr.String(), check.Equals, "")
}

func (s *KanXCmdProcessClientSuite) TestProcessClientGet_1(c *check.C) {
	addr := c.MkDir() + "/kanister.sock"
	ctx, can := context.WithCancel(context.Background())
	defer can()
	go func() {
		err := startServer(ctx, addr)
		c.Assert(err, check.IsNil)
	}()
	err := waitSock(ctx, addr)
	c.Assert(err, check.IsNil)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "create", "echo", "hello world")
	c.Assert(err, check.IsNil)
	pr := &ProcessResult{}
	err = json.Unmarshal(stdout.Bytes(), pr)
	c.Assert(err, check.IsNil)
	c.Assert(pr.Pid, Not(Equals), "")
	c.Assert(pr.State, check.Equals, "PROCESS_STATE_RUNNING")
	c.Assert(stderr.String(), check.Equals, "")
	// get output
	resetBuffers(stdout, stderr)
	err = executeCommand(ctx, stdout, stderr, "process", "client", "-a", addr, "output", pr.Pid)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "hello world\n")
	c.Assert(stderr.String(), check.Equals, "")
	resetBuffers(stdout, stderr)
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "get", "555555555")
	c.Assert(err, NotNil)
	c.Assert(stderr.String(), Matches, ".*Process not found.*\n")
}
