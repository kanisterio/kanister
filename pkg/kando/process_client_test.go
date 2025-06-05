package kando

import (
	"bytes"
	"context"
	"encoding/json"

	v1 "gopkg.in/check.v1"
)

type KanXCmdProcessClientSuite struct{}

var _ = v1.Suite(&KanXCmdProcessClientSuite{})

func (s *KanXCmdProcessClientSuite) TestProcessClientCreate(c *v1.C) {
	addr := c.MkDir() + "/kanister.sock"
	ctx, can := context.WithCancel(context.Background())
	defer can()
	go func() {
		err := startServer(ctx, addr)
		c.Assert(err, v1.IsNil)
	}()
	err := waitSock(ctx, addr)
	c.Assert(err, v1.IsNil)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "create", "sleep", "2s")
	c.Assert(err, v1.IsNil)
	pr := &ProcessResult{}
	err = json.Unmarshal(stdout.Bytes(), pr)
	c.Assert(err, v1.IsNil)
	c.Assert(stderr.String(), v1.Equals, "")
	// get output
	resetBuffers(stdout, stderr)
	err = executeCommand(ctx, stdout, stderr, "process", "client", "-a", addr, "output", pr.Pid)
	c.Assert(err, v1.IsNil)
	c.Assert(stdout.String(), v1.Equals, "")
	c.Assert(stderr.String(), v1.Equals, "")
}

func (s *KanXCmdProcessClientSuite) TestProcessClientList_0(c *v1.C) {
	addr := c.MkDir() + "/kanister.sock"
	ctx, can := context.WithCancel(context.Background())
	defer can()
	go func() {
		err := startServer(ctx, addr)
		c.Assert(err, v1.IsNil)
	}()
	err := waitSock(ctx, addr)
	c.Assert(err, v1.IsNil)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "list")
	c.Assert(err, v1.IsNil)
	pr := &ProcessResult{}
	dcd := json.NewDecoder(stdout)
	n := 0
	for dcd.More() {
		err = dcd.Decode(pr)
		n++
	}
	c.Assert(err, v1.IsNil)
	c.Assert(n, v1.Equals, 0)
	c.Assert(stderr.String(), v1.Equals, "")
}

func (s *KanXCmdProcessClientSuite) TestProcessClientList_1(c *v1.C) {
	addr := c.MkDir() + "/kanister.sock"
	ctx, can := context.WithCancel(context.Background())
	defer can()
	go func() {
		err := startServer(ctx, addr)
		c.Assert(err, v1.IsNil)
	}()
	err := waitSock(ctx, addr)
	c.Assert(err, v1.IsNil)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "create", "sleep", "2s")
	c.Assert(err, v1.IsNil)
	resetBuffers(stdout, stderr)
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "create", "sleep", "2s")
	c.Assert(err, v1.IsNil)
	resetBuffers(stdout, stderr)
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "list")
	c.Assert(err, v1.IsNil)
	pr := &ProcessResult{}
	dcd := json.NewDecoder(stdout)
	n := 0
	for dcd.More() {
		err = dcd.Decode(pr)
		n++
	}
	c.Assert(err, v1.IsNil)
	c.Assert(n, v1.Equals, 2)
	c.Assert(stderr.String(), v1.Equals, "")
}

func (s *KanXCmdProcessClientSuite) TestProcessClientSignal(c *v1.C) {
	addr := c.MkDir() + "/kanister.sock"
	ctx, can := context.WithCancel(context.Background())
	defer can()
	go func() {
		err := startServer(ctx, addr)
		c.Assert(err, v1.IsNil)
	}()
	err := waitSock(ctx, addr)
	c.Assert(err, v1.IsNil)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "create", "sleep", "2s")
	c.Assert(err, v1.IsNil)
	pr := &ProcessResult{}
	err = json.Unmarshal(stdout.Bytes(), pr)
	c.Assert(err, v1.IsNil)
	c.Assert(pr.State, v1.Equals, "PROCESS_STATE_RUNNING")
	c.Assert(stderr.String(), v1.Equals, "")
	resetBuffers(stdout, stderr)
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "signal", pr.Pid, "2")
	c.Assert(err, v1.IsNil)
	err = json.Unmarshal(stdout.Bytes(), pr)
	c.Assert(err, v1.IsNil)
	c.Assert(pr.State, v1.Equals, "PROCESS_STATE_RUNNING")
	c.Assert(stderr.String(), v1.Equals, "")
	resetBuffers(stdout, stderr)
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "output", pr.Pid)
	c.Assert(err, v1.IsNil)
	c.Assert(stdout.String(), v1.Equals, "")
	c.Assert(stderr.String(), v1.Equals, "")
	resetBuffers(stdout, stderr)
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "get", pr.Pid)
	c.Assert(err, v1.IsNil)
	err = json.Unmarshal(stdout.Bytes(), pr)
	c.Assert(err, v1.IsNil)
	c.Assert(pr.State, v1.Equals, "PROCESS_STATE_FAILED")
	c.Assert(stderr.String(), v1.Equals, "")
}

// TestProcessClientOutput check that output command outputs stdout and stderr to their respective FDs.
func (s *KanXCmdProcessClientSuite) TestProcessClientOutput(c *v1.C) {
	addr := c.MkDir() + "/kanister.sock"
	ctx, can := context.WithCancel(context.Background())
	defer can()
	go func() {
		err := startServer(ctx, addr)
		c.Assert(err, v1.IsNil)
	}()
	err := waitSock(ctx, addr)
	c.Assert(err, v1.IsNil)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "create", "--", "bash", "-c", "echo 'hello world 1' && echo 'hello world 2' 1>&2")
	c.Assert(err, v1.IsNil)
	pr := &ProcessResult{}
	err = json.Unmarshal(stdout.Bytes(), pr)
	c.Assert(err, v1.IsNil)
	c.Assert(stderr.String(), v1.Equals, "")
	// get output
	resetBuffers(stdout, stderr)
	err = executeCommand(ctx, stdout, stderr, "process", "client", "-a", addr, "output", pr.Pid)
	c.Assert(err, v1.IsNil)
	c.Assert(stdout.String(), v1.Equals, "hello world 1\n")
	c.Assert(stderr.String(), v1.Equals, "hello world 2\n")
}

// TestProcessClientExecute_RedirectStdout checks that stdout contains JSON process metadata and process output without additional output from logging.
func (s *KanXCmdProcessClientSuite) TestProcessClientExecute_RedirectStdout(c *v1.C) {
	addr := c.MkDir() + "/kanister.sock"
	ctx, can := context.WithCancel(context.Background())
	defer can()
	go func() {
		err := startServer(ctx, addr)
		c.Assert(err, v1.IsNil)
	}()
	err := waitSock(ctx, addr)
	c.Assert(err, v1.IsNil)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "execute", "echo", "hello world")
	c.Assert(err, v1.IsNil)
	bs := stdout.Bytes()
	pr := &ProcessResult{}
	dc := json.NewDecoder(bytes.NewReader(bs))
	err = dc.Decode(pr)
	c.Assert(err, v1.IsNil)
	c.Assert(dc.More(), v1.Equals, true)
	rest := dc.InputOffset()
	c.Assert(string(bs[rest:]), v1.Equals, "hello world\n")
	c.Assert(stderr.String(), v1.Equals, "")
}

// TestProcessClientExecute_RedirectStderr checks that stderr without additional output from logging.
func (s *KanXCmdProcessClientSuite) TestProcessClientExecute_RedirectStderr(c *v1.C) {
	addr := c.MkDir() + "/kanister.sock"
	ctx, can := context.WithCancel(context.Background())
	defer can()
	go func() {
		err := startServer(ctx, addr)
		c.Assert(err, v1.IsNil)
	}()
	err := waitSock(ctx, addr)
	c.Assert(err, v1.IsNil)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "execute", "--", "bash", "-c", "echo 'hello world' 1>&2")
	c.Assert(err, v1.IsNil)
	bs := stdout.Bytes()
	pr := &ProcessResult{}
	dc := json.NewDecoder(bytes.NewReader(bs))
	err = dc.Decode(pr)
	c.Assert(err, v1.IsNil)
	c.Assert(stderr.String(), v1.Equals, "hello world\n")
}

func (s *KanXCmdProcessClientSuite) TestProcessClientGet_0(c *v1.C) {
	addr := c.MkDir() + "/kanister.sock"
	ctx, can := context.WithCancel(context.Background())
	defer can()
	go func() {
		err := startServer(ctx, addr)
		c.Assert(err, v1.IsNil)
	}()
	err := waitSock(ctx, addr)
	c.Assert(err, v1.IsNil)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "create", "echo", "hello world")
	c.Assert(err, v1.IsNil)
	pr := &ProcessResult{}
	err = json.Unmarshal(stdout.Bytes(), pr)
	c.Assert(err, v1.IsNil)
	c.Assert(pr.Pid, v1.Not(v1.Equals), "")
	c.Assert(pr.State, v1.Equals, "PROCESS_STATE_RUNNING")
	c.Assert(stderr.String(), v1.Equals, "")
	// get output
	resetBuffers(stdout, stderr)
	err = executeCommand(ctx, stdout, stderr, "process", "client", "-a", addr, "output", pr.Pid)
	c.Assert(err, v1.IsNil)
	c.Assert(stdout.String(), v1.Equals, "hello world\n")
	c.Assert(stderr.String(), v1.Equals, "")
	resetBuffers(stdout, stderr)
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "get", pr.Pid)
	c.Assert(err, v1.IsNil)
	pr = &ProcessResult{}
	err = json.Unmarshal(stdout.Bytes(), pr)
	c.Assert(err, v1.IsNil)
	c.Assert(pr.Pid, v1.Not(v1.Equals), "")
	c.Assert(pr.State, v1.Equals, "PROCESS_STATE_SUCCEEDED")
	c.Assert(stderr.String(), v1.Equals, "")
}

func (s *KanXCmdProcessClientSuite) TestProcessClientGet_1(c *v1.C) {
	addr := c.MkDir() + "/kanister.sock"
	ctx, can := context.WithCancel(context.Background())
	defer can()
	go func() {
		err := startServer(ctx, addr)
		c.Assert(err, v1.IsNil)
	}()
	err := waitSock(ctx, addr)
	c.Assert(err, v1.IsNil)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "create", "echo", "hello world")
	c.Assert(err, v1.IsNil)
	pr := &ProcessResult{}
	err = json.Unmarshal(stdout.Bytes(), pr)
	c.Assert(err, v1.IsNil)
	c.Assert(pr.Pid, v1.Not(v1.Equals), "")
	c.Assert(pr.State, v1.Equals, "PROCESS_STATE_RUNNING")
	c.Assert(stderr.String(), v1.Equals, "")
	// get output
	resetBuffers(stdout, stderr)
	err = executeCommand(ctx, stdout, stderr, "process", "client", "-a", addr, "output", pr.Pid)
	c.Assert(err, v1.IsNil)
	c.Assert(stdout.String(), v1.Equals, "hello world\n")
	c.Assert(stderr.String(), v1.Equals, "")
	resetBuffers(stdout, stderr)
	err = executeCommand(ctx, stdout, stderr, "process", "client", "--as-json", "-a", addr, "get", "555555555")
	c.Assert(err, v1.NotNil)
	c.Assert(stderr.String(), v1.Matches, ".*Process not found.*\n")
}
