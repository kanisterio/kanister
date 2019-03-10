package awsebs

import (
	"context"
	"testing"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type AWSEBSSuite struct{}

var _ = Suite(&AWSEBSSuite{})

func (s AWSEBSSuite) TestQueryRegionToZones(c *C) {
	c.Skip("Only works on AWS")
	ctx := context.Background()
	region := "us-east-1"
	zs, err := queryRegionToZones(ctx, region)
	c.Assert(err, IsNil)
	c.Assert(zs, DeepEquals, []string{"us-east-1a", "us-east-1b", "us-east-1c", "us-east-1d", "us-east-1e", "us-east-1f"})
}
