package tags

import (
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type TagsSuite struct{}

var _ = Suite(&TagsSuite{})

func (s *TagsSuite) TestSanitizeValueForGCP(c *C) {
	for _, tc := range []struct {
		input  string
		output string
	}{
		{
			input:  "abcd",
			output: "abcd",
		},
		{
			input:  "kasten__snapshot-wordpress-on-rbd-ceph-ns-2021-04-15t18-11-27z-abcd",
			output: "kasten__snapshot-wordpress-on-rbd-ceph-ns-2021-04-15t18-11-27z",
		},
		{
			input:  "kasten__snapshot-wordpress-on-rbd-ceph-ns-",
			output: "kasten__snapshot-wordpress-on-rbd-ceph-ns",
		},
		{
			input:  "kasten__snapshot-wordpress-on-rbd-ceph-ns---",
			output: "kasten__snapshot-wordpress-on-rbd-ceph-ns",
		},
		{
			input:  "kasten__snapshot-wordpress! ?*()",
			output: "kasten__snapshot-wordpress",
		},
		{
			input:  "kasten__snapshot-wordpress-on-rbd-ceph-ns-__",
			output: "kasten__snapshot-wordpress-on-rbd-ceph-ns",
		},
		{
			input:  "kasten__snapshot-wordpress-on-rbd-ceph-ns__--",
			output: "kasten__snapshot-wordpress-on-rbd-ceph-ns",
		},
		{
			input:  "ALLCAPS",
			output: "allcaps",
		},
	} {
		out := SanitizeValueForGCP(tc.input)
		c.Assert(out, Equals, tc.output)
	}
}
