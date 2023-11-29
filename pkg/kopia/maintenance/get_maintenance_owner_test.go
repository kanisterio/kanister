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
	"encoding/json"
	"testing"
	"time"

	kopiacli "github.com/kopia/kopia/cli"
	"github.com/kopia/kopia/repo/maintenance"
	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func TestKopiaMaintenanceWrappers(t *testing.T) { TestingT(t) }

type KopiaMaintenanceOwnerTestSuite struct{}

var _ = Suite(&KopiaMaintenanceOwnerTestSuite{})

func (kMaintenanceOwner *KopiaMaintenanceOwnerTestSuite) TestParseMaintenanceOwnerOutput(c *C) {
	maintInfoResult := kopiacli.MaintenanceInfo{
		Params: maintenance.Params{
			Owner: "owner@hostname",
			QuickCycle: maintenance.CycleParams{
				Enabled:  false,
				Interval: 10 * time.Minute,
			},
			FullCycle: maintenance.CycleParams{
				Enabled:  true,
				Interval: 10 * time.Minute,
			},
		},
		Schedule: maintenance.Schedule{
			NextFullMaintenanceTime:  time.Now().Add(1 * time.Hour),
			NextQuickMaintenanceTime: time.Now().Add(10 * time.Minute),
			Runs: map[maintenance.TaskType][]maintenance.RunInfo{
				"asdf": {
					{
						Start:   time.Now().Add(-10 * time.Minute),
						End:     time.Now().Add(-5 * time.Minute),
						Success: true,
						Error:   "",
					},
				},
				"bsdf": {
					{
						Start:   time.Now().Add(-100 * time.Minute),
						End:     time.Now().Add(-50 * time.Minute),
						Success: false,
						Error:   "some error",
					},
				},
			},
		},
	}
	maintOutput, err := json.Marshal(maintInfoResult)
	c.Assert(err, IsNil)

	for _, tc := range []struct {
		desc          string
		output        []byte
		expectedOwner string
		expectedErr   Checker
	}{
		{
			desc:          "empty output",
			output:        []byte{},
			expectedOwner: "",
			expectedErr:   NotNil,
		},
		{
			desc:          "maintenance output",
			output:        maintOutput,
			expectedOwner: "owner@hostname",
			expectedErr:   IsNil,
		},
	} {
		owner, err := parseOwner(tc.output)
		c.Assert(err, tc.expectedErr, Commentf("Case: %s", tc.desc))
		c.Assert(owner, Equals, tc.expectedOwner, Commentf("Case: %s", tc.desc))
	}
}
