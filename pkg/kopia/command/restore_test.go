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

type KopiaRestoreTestSuite struct{}

var _ = Suite(&KopiaRestoreTestSuite{})

func (kRestore *KopiaRestoreTestSuite) TestRestoreCommands(c *C) {
	for _, tc := range []struct {
		f           func() []string
		expectedLog string
	}{
		{
			f: func() []string {
				args := RestoreCommandArgs{
					CommandArgs: &CommandArgs{
						RepoPassword:   "encr-key",
						ConfigFilePath: "path/kopia.config",
						LogDirectory:   "cache/log",
					},
					RootID:      "snapshot-id",
					TargetPath:  "target/path",
					Parallelism: 8,
				}
				return Restore(args)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=encr-key restore snapshot-id target/path --no-ignore-permission-errors --parallel=8",
		},
		{
			f: func() []string {
				args := RestoreCommandArgs{
					CommandArgs: &CommandArgs{
						RepoPassword:   "encr-key",
						ConfigFilePath: "path/kopia.config",
						LogDirectory:   "cache/log",
					},
					RootID:     "snapshot-id",
					TargetPath: "target/path",
				}
				return Restore(args)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=encr-key restore snapshot-id target/path --no-ignore-permission-errors",
		},
		{
			f: func() []string {
				args := RestoreCommandArgs{
					CommandArgs: &CommandArgs{
						RepoPassword:   "encr-key",
						ConfigFilePath: "path/kopia.config",
						LogDirectory:   "cache/log",
					},
					RootID:                 "snapshot-id",
					TargetPath:             "target/path",
					IgnorePermissionErrors: true,
					Parallelism:            32,
				}
				return Restore(args)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=encr-key restore snapshot-id target/path --ignore-permission-errors --parallel=32",
		},
	} {
		cmd := strings.Join(tc.f(), " ")
		c.Check(cmd, Equals, tc.expectedLog)
	}
}
