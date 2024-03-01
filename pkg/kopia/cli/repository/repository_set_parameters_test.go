// Copyright 2024 The Kanister Authors.
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

package repository

import (
	"testing"

	"github.com/kanisterio/safecli"
	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
)

func TestRepositorySetParametersCommand(t *testing.T) { check.TestingT(t) }

// Test Repository Set Parameters command
var _ = check.Suite(test.NewCommandSuite([]test.CommandTest{
	{
		Name: "repository set-parameters with default retention",
		Command: func() (*safecli.Builder, error) {
			args := SetParametersArgs{
				Common: common,
			}
			return SetParameters(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-dir=cache/log",
			"--log-level=error",
			"--password=encr-key",
			"repository",
			"set-parameters",
		},
	},
	{
		Name: "repository set-parameters with custom retention args",
		Command: func() (*safecli.Builder, error) {
			args := SetParametersArgs{
				Common:          common,
				RetentionMode:   retentionMode,
				RetentionPeriod: retentionPeriod,
			}
			return SetParameters(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-dir=cache/log",
			"--log-level=error",
			"--password=encr-key",
			"repository",
			"set-parameters",
			"--retention-mode=Locked",
			"--retention-period=15m0s",
		},
	},
	{
		Name: "repository set-parameters with custom retention mode only",
		Command: func() (*safecli.Builder, error) {
			args := SetParametersArgs{
				Common:        common,
				RetentionMode: retentionMode,
			}
			return SetParameters(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-dir=cache/log",
			"--log-level=error",
			"--password=encr-key",
			"repository",
			"set-parameters",
			"--retention-mode=Locked",
			"--retention-period=0s",
		},
	},
}))
