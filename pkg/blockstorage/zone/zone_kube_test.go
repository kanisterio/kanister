package zone

import (
	"context"

	. "gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/kube"
)

type KubeTestZoneSuite struct{}

var _ = Suite(&KubeTestZoneSuite{})

func (s KubeTestZoneSuite) TestNodeZones(c *C) {
	c.Skip("Fails in Minikube")
	ctx := context.Background()
	cli, err := kube.NewClient()
	c.Assert(err, IsNil)
	zones, _, err := NodeZonesAndRegion(ctx, cli)
	c.Assert(err, IsNil)
	c.Assert(zones, Not(HasLen), 0)
}
