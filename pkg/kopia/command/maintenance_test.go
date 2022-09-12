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
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/kanisterio/kanister/pkg/logsafe"
)

func TestMaintenanceCommandsLogging(t *testing.T) {
	c := qt.New(t)

	for _, tc := range []struct {
		f           func() logsafe.Cmd
		expectedLog string
	}{
		{
			f: func() logsafe.Cmd {
				args := MaintenanceSetOwnerCommandArgs{
					CommandArgs: &CommandArgs{
						EncryptionKey:  "encr-key",
						ConfigFilePath: "path/kopia.config",
						LogDirectory:   "cache/log",
					},
					CustomOwner: "username@hostname",
				}
				return maintenanceSetOwnerCommand(args)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=<****> maintenance set --owner=username@hostname",
		},
		{
			f: func() logsafe.Cmd {
				args := MaintenanceRunCommandArgs{
					CommandArgs: &CommandArgs{
						EncryptionKey:  "encr-key",
						ConfigFilePath: "path/kopia.config",
						LogDirectory:   "cache/log",
					},
				}
				return maintenanceRunCommand(args)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=<****> maintenance run",
		},
		{
			f: func() logsafe.Cmd {
				args := MaintenanceInfoCommandArgs{
					CommandArgs: &CommandArgs{
						EncryptionKey:  "encr-key",
						ConfigFilePath: "path/kopia.config",
						LogDirectory:   "cache/log",
					},
					GetJsonOutput: false,
				}
				return maintenanceInfoCommand(args)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=<****> maintenance info",
		},
		{
			f: func() logsafe.Cmd {
				args := MaintenanceInfoCommandArgs{
					CommandArgs: &CommandArgs{
						EncryptionKey:  "encr-key",
						ConfigFilePath: "path/kopia.config",
						LogDirectory:   "cache/log",
					},
					GetJsonOutput: true,
				}
				return maintenanceInfoCommand(args)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=<****> maintenance info --json",
		},
	} {
		cmd := tc.f()
		c.Check(cmd.String(), qt.Equals, tc.expectedLog)
	}
}
