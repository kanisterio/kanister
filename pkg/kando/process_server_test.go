package kando

import (
	"context"
	"testing"
	"time"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type KanXCmdProcessServerSuite struct{}

var _ = Suite(&KanXCmdProcessServerSuite{})

func (s *KanXCmdProcessServerSuite) TestProcessServer(c *C) {
	addr := c.MkDir() + "/kanister.sock"
	ctx, can := context.WithTimeout(context.Background(), time.Second)
	rc := newRootCommand()
	rc.SetArgs([]string{"process", "server", "-a", addr})
	go func() {
		err := rc.ExecuteContext(ctx)
		// err does not matter, but let's log it anyway.
		if err != nil {
			c.Log(err)
		}
	}()
	err := waitSock(ctx, addr)
	c.Assert(err, IsNil)
	can()
}
