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

	"github.com/kanisterio/kanister/pkg/blockstorage/utils/zone"
)

type ZoneSuite struct{}

var _ = Suite(&ZoneSuite{})

func (s ZoneSuite) TestZoneWithUnknownNodeZones(c *C) {
	defaultZones := []string{"us-west-2a", "us-west-2b", "us-west-2c"}
	for _, tc := range []struct {
		zones []string
		in    map[string]struct{}
		out   map[string]struct{}
	}{
		{
			zones: defaultZones,
			in:    map[string]struct{}{"us-west-2a": struct{}{}},
			out:   map[string]struct{}{"us-west-2a": struct{}{}},
		},
		{
			zones: defaultZones,
			in:    map[string]struct{}{"us-east-1f": struct{}{}},
			out:   map[string]struct{}{"us-west-2a": struct{}{}},
		},
		{
			zones: defaultZones,
			in:    map[string]struct{}{"us-east-2b": struct{}{}},
			out:   map[string]struct{}{"us-west-2b": struct{}{}},
		},
	} {
		z := zone.SanitizeAvailableZones(tc.in, tc.zones)
		c.Assert(z, DeepEquals, tc.out)
	}
}

var _ zone.Mapper = (*ebsTest)(nil)

type ebsTest struct{}

func (et *ebsTest) FromRegion(ctx context.Context, region string) ([]string, error) {
	// Fall back to using a static map.
	return staticRegionToZones(region)
}
