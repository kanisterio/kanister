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

package opts_test

import (
	"testing"

	"github.com/kanisterio/safecli/command"
	"github.com/kanisterio/safecli/test"
	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/kopia/cli/args"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/opts"
)

func TestCommonOptions(t *testing.T) { check.TestingT(t) }

var _ = check.Suite(&test.ArgumentSuite{Cmd: "cmd", Arguments: []test.ArgumentTest{
	{
		Name:        "LogDirectory",
		Argument:    command.NewArguments(opts.LogDirectory(""), opts.LogDirectory("/path/to/logs")),
		ExpectedCLI: []string{"cmd", "--log-dir=/path/to/logs"},
	},
	{
		Name:        "LogLevel",
		Argument:    command.NewArguments(opts.LogLevel(""), opts.LogLevel("info")),
		ExpectedCLI: []string{"cmd", "--log-level=error", "--log-level=info"},
	},
	{
		Name:        "ConfigFilePath",
		Argument:    command.NewArguments(opts.ConfigFilePath(""), opts.ConfigFilePath("/path/to/config")),
		ExpectedCLI: []string{"cmd", "--config-file=/path/to/config"},
	},
	{
		Name:        "RepoPassword",
		Argument:    command.NewArguments(opts.RepoPassword(""), opts.RepoPassword("pass123")),
		ExpectedCLI: []string{"cmd", "--password=pass123"},
	},
	{
		Name: "Common",
		Argument: opts.Common(args.Common{
			LogDirectory:   "/path/to/logs",
			LogLevel:       "trace",
			ConfigFilePath: "/path/to/config",
			RepoPassword:   "pass123",
		}),
		ExpectedCLI: []string{"cmd", "--config-file=/path/to/config", "--log-dir=/path/to/logs", "--log-level=trace", "--password=pass123"},
	},
}})
