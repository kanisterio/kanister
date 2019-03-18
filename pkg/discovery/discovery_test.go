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
