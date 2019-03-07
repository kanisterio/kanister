package zone

import (
	"testing"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type ZoneSuite struct{}

var _ = Suite(&ZoneSuite{})

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
				"zone1": {},
			},
			out: "zone1",
		},
		{
			sourceZone: "",
			nzs: map[string]struct{}{
				"zone1": {},
				"zone2": {},
			},
			out: "zone2",
		},
		{
			sourceZone: "from1",
			nzs: map[string]struct{}{
				"zone1": {},
				"zone2": {},
			},
			out: "zone1",
		},
	} {
		out, err := consistentZone(tc.sourceZone, tc.nzs)
		c.Assert(err, IsNil)
		c.Assert(out, Equals, tc.out)
	}
}
