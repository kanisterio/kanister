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

package command

import (
	"strings"

	. "gopkg.in/check.v1"
)

type KopiaPolicyTestSuite struct{}

var _ = Suite(&KopiaPolicyTestSuite{})

func (kPolicy *KopiaPolicyTestSuite) TestPolicySetCommands(c *C) {
	for _, tc := range []struct {
		f           func() []string
		expectedLog string
	}{
		{
			f: func() []string {
				args := PolicySetGlobalCommandArgs{
					CommandArgs: &CommandArgs{
						RepoPassword:   "encr-key",
						ConfigFilePath: "path/kopia.config",
						LogDirectory:   "cache/log",
					},
					Modifications: policyChanges{"asdf": "bsdf"},
				}
				return PolicySetGlobal(args)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=encr-key policy set --global asdf=bsdf",
		},
	} {
		cmd := strings.Join(tc.f(), " ")
		c.Check(cmd, Equals, tc.expectedLog)
	}
}

func (kPolicy *KopiaPolicyTestSuite) TestPolicyShowCommands(c *C) {
	for _, tc := range []struct {
		f           func() []string
		expectedLog string
	}{
		{
			f: func() []string {
				args := PolicyShowGlobalCommandArgs{
					CommandArgs: &CommandArgs{
						RepoPassword:   "encr-key",
						ConfigFilePath: "path/kopia.config",
						LogDirectory:   "cache/log",
					},
					GetJSONOutput: true,
				}
				return PolicyShowGlobal(args)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=encr-key policy show --global --json",
		},
	} {
		cmd := strings.Join(tc.f(), " ")
		c.Check(cmd, Equals, tc.expectedLog)
	}
}
