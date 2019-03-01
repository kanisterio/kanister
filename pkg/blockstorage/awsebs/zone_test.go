package awsebs

import (
	"context"

	. "gopkg.in/check.v1"
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

		z, err := zoneWithUnknownNodeZones(ctx, tc.region, tc.in)
		c.Assert(err, IsNil)
		c.Assert(z, Not(Equals), "")
		if tc.out != "" {
			c.Assert(z, Equals, tc.out)
		}
	}
}

func (s ZoneSuite) TestConsistentZone(c *C) {
	// We don't care what the answer is as long as it's consistent.
	for _, tc := range []struct {
		sourceZone string
		nzs        map[string]struct{}
		out        string
	}{
		{
			sourceZone: "",
			nzs: map[string]struct{}{
				"zone1": struct{}{},
			},
			out: "zone1",
		},
		{
			sourceZone: "",
			nzs: map[string]struct{}{
				"zone1": struct{}{},
				"zone2": struct{}{},
			},
			out: "zone2",
		},
		{
			sourceZone: "from1",
			nzs: map[string]struct{}{
				"zone1": struct{}{},
				"zone2": struct{}{},
			},
			out: "zone1",
		},
	} {
		out, err := consistentZone(tc.sourceZone, tc.nzs)
		c.Assert(err, IsNil)
		c.Assert(out, Equals, tc.out)
	}
}
