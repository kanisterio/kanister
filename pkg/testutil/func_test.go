// Copyright 2019 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testutil

import (
	"context"
	"strings"

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

func (s *FuncSuite) TestCancelFunc(c *C) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan bool)
	go func() {
		_, err := cancelFunc(ctx, param.TemplateParams{}, nil)
		c.Assert(err, NotNil)
		c.Assert(strings.Contains(err.Error(), "context canceled"), Equals, true)
		close(done)
	}()
	c.Assert(CancelFuncStarted(), NotNil)
	select {
	case <-done:
		c.FailNow()
	default:
	}
	cancel()
	c.Assert(CancelFuncOut().Error(), DeepEquals, "context canceled")
	<-done
}
