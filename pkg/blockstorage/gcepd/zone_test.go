package gcepd

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
			region: "us-west2",
			in:     "us-west2-a",
			out:    "us-west2-a",
		},
		{
			region: "us-west2",
			in:     "us-east1-f",
			out:    "us-west2-a",
		},
		{
			region: "us-west2",
			in:     "us-east2-b",
			out:    "us-west2-b",
		},
		{
			region: "us-west2",
			in:     "us-east1-f",
			out:    "us-west2-a",
		},
	} {
		var t = &gcpTest{}
		z, err := zone.WithUnknownNodeZones(ctx, t, tc.region, tc.in)
		c.Assert(err, IsNil)
		c.Assert(z, Not(Equals), "")
		if tc.out != "" {
			c.Assert(z, Equals, tc.out)
		}
	}
}

var _ zone.Mapper = (*gcpTest)(nil)

type gcpTest struct{}

func (gt *gcpTest) FromRegion(ctx context.Context, region string) ([]string, error) {
	// Fall back to using a static map.
	return staticRegionToZones(region)
}
