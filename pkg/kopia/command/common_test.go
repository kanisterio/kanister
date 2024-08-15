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

package command

import (
	"os"

	"gopkg.in/check.v1"
)

type CommonUtilsSuite struct{}

var _ = check.Suite(&CommonUtilsSuite{})

func (s *CommonUtilsSuite) TestCommonArgs(c *check.C) {
	for _, tc := range []struct {
		setup       func() func()
		arg         *CommandArgs
		expectedCmd []string
		comment     string
	}{
		{
			setup:   func() func() { return func() {} },
			comment: "Default log settings",
			arg: &CommandArgs{
				RepoPassword:   "pass123",
				ConfigFilePath: "/tmp/config.file",
				LogDirectory:   "/tmp/log.dir",
			},
			expectedCmd: []string{"kopia",
				"--log-level=error",
				"--config-file=/tmp/config.file",
				"--log-dir=/tmp/log.dir",
				"--password=pass123",
			},
		}, {
			setup:   func() func() { return func() {} },
			comment: "Custom log level passed via args, default file log level",
			arg: &CommandArgs{
				LogLevel:       "info",
				RepoPassword:   "pass123",
				ConfigFilePath: "/tmp/config.file",
				LogDirectory:   "/tmp/log.dir",
			},
			expectedCmd: []string{"kopia",
				"--log-level=info",
				"--config-file=/tmp/config.file",
				"--log-dir=/tmp/log.dir",
				"--password=pass123",
			},
		}, {
			setup:   func() func() { return func() {} },
			comment: "Custom log level and file log level, both passed via args",
			arg: &CommandArgs{
				LogLevel:       "info",
				FileLogLevel:   "info",
				RepoPassword:   "pass123",
				ConfigFilePath: "/tmp/config.file",
				LogDirectory:   "/tmp/log.dir",
			},
			expectedCmd: []string{"kopia",
				"--log-level=info",
				"--file-log-level=info",
				"--config-file=/tmp/config.file",
				"--log-dir=/tmp/log.dir",
				"--password=pass123",
			},
		}, {
			setup: func() func() {
				origLogLevel := os.Getenv(LogLevelVarName)
				err := os.Setenv(LogLevelVarName, "debug")
				c.Assert(err, check.IsNil)

				return func() {
					err := os.Setenv(LogLevelVarName, origLogLevel)
					c.Assert(err, check.IsNil)
				}
			},
			comment: "Custom log level passed via env variable, file log level passed via args",
			arg: &CommandArgs{
				FileLogLevel:   "info",
				RepoPassword:   "pass123",
				ConfigFilePath: "/tmp/config.file",
				LogDirectory:   "/tmp/log.dir",
			},
			expectedCmd: []string{"kopia",
				"--log-level=debug",
				"--file-log-level=info",
				"--config-file=/tmp/config.file",
				"--log-dir=/tmp/log.dir",
				"--password=pass123",
			},
		}, {
			setup: func() func() {
				origLogLevel := os.Getenv(LogLevelVarName)
				origFileLogLevel := os.Getenv(FileLogLevelVarName)
				err := os.Setenv(LogLevelVarName, "debug")
				c.Assert(err, check.IsNil)
				err = os.Setenv(FileLogLevelVarName, "debug")
				c.Assert(err, check.IsNil)
				return func() {
					err := os.Setenv(LogLevelVarName, origLogLevel)
					c.Assert(err, check.IsNil)
					err = os.Setenv(FileLogLevelVarName, origFileLogLevel)
					c.Assert(err, check.IsNil)
				}
			},
			comment: "Custom log level and file log level both passed via env variable",
			arg: &CommandArgs{
				RepoPassword:   "pass123",
				ConfigFilePath: "/tmp/config.file",
				LogDirectory:   "/tmp/log.dir",
			},
			expectedCmd: []string{"kopia",
				"--log-level=debug",
				"--file-log-level=debug",
				"--config-file=/tmp/config.file",
				"--log-dir=/tmp/log.dir",
				"--password=pass123",
			},
		},
	} {
		c.Log(tc.comment)
		cleanup := tc.setup()
		defer cleanup()
		cmd := stringSliceCommand(commonArgs(tc.arg))
		c.Assert(cmd, check.DeepEquals, tc.expectedCmd)
	}
}
