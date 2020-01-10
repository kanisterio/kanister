package azure

import (
	"context"
	"testing"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type ClientSuite struct{}

var _ = Suite(&ClientSuite{})

func (s *ClientSuite) SetUpSuite(c *C) {
}

func (s *ClientSuite) TestClient(c *C) {
	c.Skip("Until Azure will be fully integrated into build.sh")
	azCli, err := NewClient(context.Background())
	c.Assert(err, IsNil)

	c.Assert(azCli.SubscriptionID, NotNil)
	c.Assert(azCli.Authorizer, NotNil)
	c.Assert(azCli.DisksClient, NotNil)
	c.Assert(azCli.SnapshotsClient, NotNil)
	_, err = azCli.DisksClient.List(context.Background())
	c.Assert(err, IsNil)
}
