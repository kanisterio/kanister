package testutil

import (
	"context"

	. "gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/param"
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
	go func() {
		_, err := failFunc(ctx, param.TemplateParams{}, nil)
		c.Assert(err, NotNil)
	}()
	c.Assert(FailFuncError().Error(), Equals, "Kanister function failed")
}

func (s *FuncSuite) TestWaitFunc(c *C) {
	ctx := context.Background()
	done := make(chan bool)
	go func() {
		_, err := waitFunc(ctx, param.TemplateParams{}, nil)
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
	args := map[string]interface{}{"arg1": []string{"foo", "bar"}}
	go func() {
		_, err := argsFunc(ctx, param.TemplateParams{}, args)
		c.Assert(err, IsNil)
	}()
	c.Assert(ArgFuncArgs(), DeepEquals, args)
}

func (s *FuncSuite) TestOutputFunc(c *C) {
	ctx := context.Background()
	args := map[string]interface{}{"arg1": []string{"foo", "bar"}}
	go func() {
		output, err := outputFunc(ctx, param.TemplateParams{}, args)
		c.Assert(err, IsNil)
		c.Assert(output, DeepEquals, args)
	}()
	c.Assert(OutputFuncOut(), DeepEquals, args)
}
