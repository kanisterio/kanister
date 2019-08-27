// Copyright 2019 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
		z := zone.WithUnknownNodeZones(ctx, t, tc.region, tc.in, make(map[string]struct{}))
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
