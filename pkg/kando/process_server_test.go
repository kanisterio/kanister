package kando

import (
	"context"
	"testing"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type KanXCmdProcessServerSuite struct{}

var _ = Suite(&KanXCmdProcessServerSuite{})

func (s *KanXCmdProcessServerSuite) TestProcessServer(c *C) {
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
