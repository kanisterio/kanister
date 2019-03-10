// +build !unit

package zone

import (
	"context"
	"fmt"

	. "gopkg.in/check.v1"
)

type KubeTestAWSEBSSuite struct{}

var _ = Suite(&KubeTestAWSEBSSuite{})

func (s KubeTestAWSEBSSuite) TestNodeZones(c *C) {
	// skipping this test since it fails on travis(minikube)
	c.Skip(fmt.Sprintf("Skipping TestNodeZones"))
	ctx := context.Background()
	zones, err := NodeZones(ctx)
	c.Assert(err, IsNil)
	c.Assert(zones, Not(HasLen), 0)
}
