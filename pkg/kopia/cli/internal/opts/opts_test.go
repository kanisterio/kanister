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

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/opts"
)

func TestOptions(t *testing.T) { check.TestingT(t) }

var _ = check.Suite(&test.ArgumentSuite{Cmd: "cmd", Arguments: []test.ArgumentTest{
	{
		Name:        "All",
		Argument:    command.NewArguments(opts.All(true), opts.All(false)),
		ExpectedCLI: []string{"cmd", "--all"},
	},
	{
		Name:        "Delta",
		Argument:    command.NewArguments(opts.Delta(true), opts.Delta(false)),
		ExpectedCLI: []string{"cmd", "--delta"},
	},
	{
		Name:        "ShowIdentical",
		Argument:    command.NewArguments(opts.ShowIdentical(true), opts.ShowIdentical(false)),
		ExpectedCLI: []string{"cmd", "--show-identical"},
	},
	{
		Name:        "Readonly",
		Argument:    command.NewArguments(opts.ReadOnly(true), opts.ReadOnly(false)),
		ExpectedCLI: []string{"cmd", "--readonly"},
	},
	{
		Name:        "CheckForUpdates",
		Argument:    command.NewArguments(opts.CheckForUpdates(true), opts.CheckForUpdates(false)),
		ExpectedCLI: []string{"cmd", "--check-for-updates", "--no-check-for-updates"},
	},
	{
		Name:        "JSON",
		Argument:    command.NewArguments(opts.JSON(true), opts.JSON(false)),
		ExpectedCLI: []string{"cmd", "--json"},
	},
	{
		Name:        "Delete",
		Argument:    command.NewArguments(opts.Delete(true), opts.Delete(false)),
		ExpectedCLI: []string{"cmd", "--delete"},
	},
}})
