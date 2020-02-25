// Copyright 2020 The Kanister Authors.
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

package zone

import (
	. "gopkg.in/check.v1"
)

type SanitizeZoneSuite struct{}

var _ = Suite(&SanitizeZoneSuite{})

func (s SanitizeZoneSuite) TestSanitizeZones(c *C) {
	for _, tc := range []struct {
		availableZones map[string]struct{}
		validZoneNames []string
		out            map[string]struct{}
	}{
		{
			availableZones: map[string]struct{}{
				"us-west1-a": {},
				"us-west1-b": {},
				"us-west1-c": {},
			},
			validZoneNames: []string{
				"us-west1-a",
				"us-west1-b",
				"us-west1-c",
			},
			out: map[string]struct{}{
				"us-west1-a": {},
				"us-west1-b": {},
				"us-west1-c": {},
			},
		},
		{
			availableZones: map[string]struct{}{
				"us-west1-a": {},
				"us-west1-b": {},
				"us-west1-c": {},
			},
			validZoneNames: []string{
				"us-west1a",
				"us-west1b",
				"us-west1c",
			},
			out: map[string]struct{}{
				"us-west1a": {},
				"us-west1b": {},
				"us-west1c": {},
			},
		},
		{
			availableZones: map[string]struct{}{
				"us-west1-a": {},
				"us-west1-b": {},
				"us-west1-c": {},
			},
			validZoneNames: []string{
				"us-west1a",
				"us-west1b",
				"us-west1c",
				"us-west1d",
			},
			out: map[string]struct{}{
				"us-west1a": {},
				"us-west1b": {},
				"us-west1c": {},
			},
		},
		{
			availableZones: map[string]struct{}{
				"us-west1-a": {},
				"us-west1-b": {},
				"us-west1-c": {},
			},
			validZoneNames: []string{
				"us-west1",
				"us-west2",
			},
			out: map[string]struct{}{
				"us-west1": {},
			},
		},
		{
			availableZones: map[string]struct{}{
				"us-west1-a": {},
				"us-west1-b": {},
				"us-west1-c": {},
			},
			validZoneNames: []string{
				"east",
				"west",
			},
			out: map[string]struct{}{
				"west": {},
			},
		},
	} {
		out := sanitizeAvailableZones(tc.availableZones, tc.validZoneNames)
		c.Assert(out, DeepEquals, tc.out)
	}
}
