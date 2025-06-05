package kando

import (
	"context"
	"testing"
	"time"

   check "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { check.TestingT(t) }

type KanXCmdProcessServerSuite struct{}

var _ = check.Suite(&KanXCmdProcessServerSuite{})

func (s *KanXCmdProcessServerSuite) TestProcessServer(c *check.C) {
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
	c.Assert(err, check.IsNil)
	can()
}
