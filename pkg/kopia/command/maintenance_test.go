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

type KopiaMaintenanceTestSuite struct{}

var _ = Suite(&KopiaMaintenanceTestSuite{})

func (kMaintenance *KopiaMaintenanceTestSuite) TestMaintenanceCommands(c *C) {
	commandArgs := &CommandArgs{
		RepoPassword:   "encr-key",
		ConfigFilePath: "path/kopia.config",
		LogDirectory:   "cache/log",
	}

	for _, tc := range []struct {
		f           func() []string
		expectedLog string
	}{
		{
			f: func() []string {
				args := MaintenanceInfoCommandArgs{
					CommandArgs:   commandArgs,
					GetJSONOutput: false,
				}
				return MaintenanceInfo(args)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=encr-key maintenance info",
		},
		{
			f: func() []string {
				args := MaintenanceInfoCommandArgs{
					CommandArgs:   commandArgs,
					GetJSONOutput: true,
				}
				return MaintenanceInfo(args)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=encr-key maintenance info --json",
		},
		{
			f: func() []string {
				args := MaintenanceSetOwnerCommandArgs{
					CommandArgs: commandArgs,
					CustomOwner: "username@hostname",
				}
				return MaintenanceSetOwner(args)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=encr-key maintenance set --owner=username@hostname",
		},
		{
			f: func() []string {
				args := MaintenanceRunCommandArgs{
					CommandArgs: commandArgs,
				}
				return MaintenanceRunCommand(args)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=encr-key maintenance run",
		},
		{
			f: func() []string {
				args := MaintenanceRunCommandArgs{
					CommandArgs: commandArgs,
				}
				args.CommandArgs.LogLevel = LogLevelInfo
				return MaintenanceRunCommand(args)
			},
			expectedLog: "kopia --log-level=info --config-file=path/kopia.config --log-dir=cache/log --password=encr-key maintenance run",
		},
		{
			f: func() []string {
				args := MaintenanceRunCommandArgs{
					CommandArgs: commandArgs,
				}
				args.CommandArgs.LogLevel = LogLevelError
				return MaintenanceRunCommand(args)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=encr-key maintenance run",
		},
	} {
		cmd := strings.Join(tc.f(), " ")
		c.Check(cmd, Equals, tc.expectedLog)
	}
}
