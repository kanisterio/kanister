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

func (s *KanXCmdSuite) TestProcessClient(c *C) {
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
	rc := newRootCommand()
	rc.SetErr(stderr)
	rc.SetOut(stdout)
	rc.SetArgs([]string{"process", "client", "--as-json", "-a", addr, "create", "sleep", "2s"})
	// create command to test
	err = rc.ExecuteContext(ctx)
	c.Assert(err, IsNil)
	stdouts := stdout.String()
	stderrs := stderr.String()
	pr := &ProcessResult{}
	err = json.Unmarshal([]byte(stdouts), pr)
	c.Assert(err, IsNil)
	c.Assert(stderrs, Equals, "")
	// get output
	stdout.Reset()
	stderr.Reset()
	rc = newRootCommand()
	rc.SetErr(stderr)
	rc.SetOut(stdout)
	rc.SetArgs([]string{"process", "client", "-a", addr, "output", pr.Pid})
	err = rc.ExecuteContext(ctx)
	c.Assert(err, IsNil)
	stdouts = stdout.String()
	stderrs = stderr.String()
	c.Assert(stdouts, Equals, "")
	c.Assert(stderrs, Equals, "")
}
