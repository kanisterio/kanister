package testutil

import (
	"context"

	. "gopkg.in/check.v1"
)

type FuncSuite struct {
}

var _ = Suite(&FuncSuite{})

func (s *FuncSuite) SetUpSuite(c *C) {
}

func (s *FuncSuite) TearDownSuite(c *C) {
}

func (s *FuncSuite) TestFailFunc(c *C) {
	ctx := context.Background()
	err := failFunc(ctx)
	c.Assert(err, NotNil)
}

func (s *FuncSuite) TestWaitFunc(c *C) {
	ctx := context.Background()
	done := make(chan bool)
	go func() {
		err := waitFunc(ctx)
		c.Assert(err, IsNil)
		close(done)
	}()
	select {
	case <-done:
		c.FailNow()
	default:
	}
	ReleaseWaitFunc()
	<-done
}

func (s *FuncSuite) TestArgsFunc(c *C) {
	ctx := context.Background()
	args := []string{"foo", "bar"}
	go func() {
		err := argsFunc(ctx, args...)
		c.Assert(err, IsNil)
	}()
	c.Assert(ArgFuncArgs(), DeepEquals, args)
}
