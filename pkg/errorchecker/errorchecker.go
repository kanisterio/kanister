// Copyright 2024 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package errorchecker

import (
	"regexp"

	"gopkg.in/check.v1"
)

// AssertErrorMessage is purposed to verify that error message matches wanted pattern
func AssertErrorMessage(c *check.C, err error, wanted string) {
	matches, err := regexp.MatchString("^"+wanted+"$", err.Error())
	c.Assert(err, check.IsNil)
	c.Assert(matches, check.Equals, true)
}
