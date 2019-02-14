// +build !unit

package awsebs

import (
	"context"

	. "gopkg.in/check.v1"
)

type KubeTestAWSEBSSuite struct{}

var _ = Suite(&KubeTestAWSEBSSuite{})

func (s KubeTestAWSEBSSuite) TestNodeZones(c *C) {
	ctx := context.Background()
	zones, err := nodeZones(ctx)
	c.Assert(err, IsNil)
	c.Assert(zones, Not(HasLen), 0)
}
