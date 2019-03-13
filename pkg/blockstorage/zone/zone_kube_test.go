// +build !unit

package zone

import (
	"context"

	. "gopkg.in/check.v1"
)

type KubeTestZoneSuite struct{}

var _ = Suite(&KubeTestZoneSuite{})

func (s KubeTestZoneSuite) TestNodeZones(c *C) {
	ctx := context.Background()
	zones, err := nodeZones(ctx)
	c.Assert(err, IsNil)
	c.Assert(zones, Not(HasLen), 0)
}
