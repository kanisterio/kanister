// Copyright 2022 The Kanister Authors.
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

package utils_test

import (
	"testing"
	"time"

	qt "github.com/frankban/quicktest"

	"github.com/kanisterio/kanister/pkg/utils"
)

func TestRoundUpDuration(t *testing.T) {
	c := qt.New(t)

	for _, test := range []struct {
		duration time.Duration
		rounded  time.Duration
		name     string
	}{
		{
			duration: 35 * time.Second,
			rounded:  35 * time.Second,
			name:     "round to seconds",
		},
		{
			duration: 35*time.Minute + 35*time.Second,
			rounded:  36 * time.Minute,
			name:     "round to minutes",
		},
		{
			duration: 35*time.Hour + 35*time.Minute + 35*time.Second,
			rounded:  36 * time.Hour,
			name:     "round to hours",
		},
	} {
		c.Run(test.name, func(c *qt.C) {
			c.Assert(utils.RoundUpDuration(test.duration), qt.Equals, test.rounded)
		})
	}
}

func TestDurationToString(t *testing.T) {
	c := qt.New(t)

	for _, test := range []struct {
		duration    time.Duration
		stringified string
		name        string
	}{
		{
			duration:    1 * time.Minute,
			stringified: "1m",
			name:        "just minutes",
		},
		{
			duration:    1*time.Hour + 1*time.Minute,
			stringified: "1h1m",
			name:        "hours and minutes",
		},
		{
			duration:    3 * time.Hour,
			stringified: "3h",
			name:        "just hours",
		},
		{
			duration:    24*time.Hour + 35*time.Minute + 45*time.Second,
			stringified: "24h35m45s",
			name:        "hours minutes and seconds",
		},
	} {
		c.Run(test.name, func(c *qt.C) {
			c.Assert(utils.DurationToString(test.duration), qt.Equals, test.stringified)
		})
	}
}
