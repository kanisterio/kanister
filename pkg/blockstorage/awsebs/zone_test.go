package awsebs

import (
	"context"

	. "gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/blockstorage/zone"
)

type ZoneSuite struct{}

var _ = Suite(&ZoneSuite{})

func (s ZoneSuite) TestZoneWithUnknownNodeZones(c *C) {
	ctx := context.Background()
	for _, tc := range []struct {
		region string
		in     string
		out    string
	}{
		{
			region: "us-west-2",
			in:     "us-west-2a",
			out:    "us-west-2a",
		},
		{
			region: "us-west-2",
			in:     "us-east-1f",
			out:    "us-west-2a",
		},
		{
			region: "us-west-2",
			in:     "us-east-2b",
			out:    "us-west-2b",
		},
		{
			region: "us-west-2",
			in:     "us-east-1f",
			out:    "us-west-2a",
		},
	} {
		var t = &ebsTest{}
		z, err := zone.WithUnknownNodeZones(ctx, t, tc.region, tc.in)
		c.Assert(err, IsNil)
		c.Assert(z, Not(Equals), "")
		if tc.out != "" {
			c.Assert(z, Equals, tc.out)
		}
	}
}

var _ zone.Mapper = (*ebsTest)(nil)

type ebsTest struct{}

func (et *ebsTest) FromRegion(ctx context.Context, region string) ([]string, error) {
	// Fall back to using a static map.
	return staticRegionToZones(region)
}
