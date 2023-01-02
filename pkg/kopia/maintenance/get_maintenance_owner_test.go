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

package maintenance

import (
	"testing"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func TestKopiaMaintenanceWrappers(t *testing.T) { TestingT(t) }

type KopiaMaintenanceOwnerTestSuite struct{}

var _ = Suite(&KopiaMaintenanceOwnerTestSuite{})

func (kMaintenanceOwner *KopiaMaintenanceOwnerTestSuite) TestParseMaintenanceOwnerOutput(c *C) {
	for _, tc := range []struct {
		output        string
		expectedOwner string
	}{
		{
			output:        "",
			expectedOwner: "",
		},
		{
			output: `Owner: username@hostname
			Quick Cycle:
			  scheduled: true
			  interval: 1h0m0s
			  next run: now
			Full Cycle:
			  scheduled: false
			Recent Maintenance Runs:
			`,
			expectedOwner: "username@hostname",
		},
	} {
		owner := parseOutput(tc.output)
		c.Assert(owner, Equals, tc.expectedOwner)
	}
}
