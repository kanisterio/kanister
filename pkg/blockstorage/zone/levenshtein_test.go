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

type LevenshteinSuite struct{}

var _ = Suite(&LevenshteinSuite{})

func (s LevenshteinSuite) TestLevenshteinMatch(c *C) {
	for _, tc := range []struct {
		input   string
		options []string
		out     string
	}{
		{
			input: "us-west1-a",
			options: []string{
				"us-west1-a",
				"us-west1-b",
				"us-west1-c",
			},
			out: "us-west1-a",
		},
		{
			input: "us-west1-a",
			options: []string{
				"us-west1a",
				"us-west1b",
				"us-west1c",
			},
			out: "us-west1a",
		},
		{
			input: "us-west1-a",
			options: []string{
				"us-west1",
				"us-west2",
			},
			out: "us-west1",
		},
		{
			input: "us-west1-a",
			options: []string{
				"east",
				"west",
			},
			out: "west",
		},
	} {
		out := levenshteinMatch(tc.input, tc.options)
		c.Assert(out, Equals, tc.out)
	}
}
