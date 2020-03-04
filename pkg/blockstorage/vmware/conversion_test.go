package vmware

import (
	. "gopkg.in/check.v1"
)

type VMWareConversionSuite struct{}

var _ = Suite(&VMWareConversionSuite{})

func (s *VMWareConversionSuite) TestSnapshotIDConversion(c *C) {
	for _, tc := range []struct {
		fullID   string
		errCheck Checker
	}{
		{
			fullID:   "1234-abcd-5678-9213:3413-abcd-1234-1234",
			errCheck: IsNil,
		},
		{
			fullID:   "1234-abcd-5678-9213:",
			errCheck: NotNil,
		},
		{
			fullID:   ":3413-abcd-1234-1234",
			errCheck: NotNil,
		},
		{
			fullID:   "1234-abcd-5678-9213",
			errCheck: NotNil,
		},
	} {
		volID, snapID, err := splitSnapshotFullID(tc.fullID)
		c.Check(err, tc.errCheck)
		if tc.errCheck == IsNil {
			fullID := snapshotFullID(volID, snapID)
			c.Check(tc.fullID, Equals, fullID)
		}
	}
}
