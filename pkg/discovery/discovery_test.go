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

package discovery

import (
	"context"
	"testing"

	. "gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/kube"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type DiscoverSuite struct{}

var _ = Suite(&DiscoverSuite{})

func (s *DiscoverSuite) TestDiscover(c *C) {
	ctx := context.Background()
	cli, err := kube.NewClient()
	c.Assert(err, IsNil)
	gvrs, err := AllGVRs(ctx, cli.Discovery())
	c.Assert(err, IsNil)
	c.Assert(gvrs, Not(HasLen), 0)
	for _, gvr := range gvrs {
		c.Assert(gvr.Empty(), Equals, false)
		c.Assert(gvr.Version, Not(Equals), "")
		c.Assert(gvr.Resource, Not(Equals), "")
	}

	gvrs, err = NamespacedGVRs(ctx, cli.Discovery())
	c.Assert(err, IsNil)
	c.Assert(gvrs, Not(HasLen), 0)
	for _, gvr := range gvrs {
		c.Assert(gvr.Empty(), Equals, false)
		c.Assert(gvr.Version, Not(Equals), "")
		c.Assert(gvr.Resource, Not(Equals), "")
	}
}
