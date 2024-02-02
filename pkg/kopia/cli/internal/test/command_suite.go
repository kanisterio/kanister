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

package test

import (
	"fmt"

	"gopkg.in/check.v1"

	"github.com/pkg/errors"

	"github.com/kanisterio/safecli"

	"github.com/kanisterio/kanister/pkg/log"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
)

var CommonArgs = cli.CommonArgs{
	RepoPassword:   "encr-key",
	ConfigFilePath: "path/kopia.config",
	LogDirectory:   "cache/log",
}

// CommandTest defines a single test for a command.
type CommandTest struct {
	// Name of the test. (required)
	Name string

	// CLI to test. (required)
	CLI func() (safecli.CommandBuilder, error)

	// Expected CLI arguments. (optional)
	ExpectedCLI []string

	// Expected log output. (optional)
	// if empty, it will be set to ExpectedCLI joined with space.
	// if empty and ExpectedCLI is empty, it will be ignored.
	ExpectedLog string

	// Expected error. (optional)
	// If nil, no error is expected and
	// ExpectedCLI and ExpectedLog are checked.
	ExpectedErr error

	// LoggerRegex is a list of regular expressions to match against the log output. (optional)
	Logger      log.Logger
	LoggerRegex []string
}

// CommandSuite defines a test suite for commands.
type CommandSuite struct {
	Tests []CommandTest
}

// TestCommands runs all tests in the suite.
func (s *CommandSuite) TestCommands(c *check.C) {
	for _, test := range s.Tests {
		cmt := check.Commentf("FAIL: %v", test.Name)
		b, err := test.CLI()
		if test.ExpectedErr == nil {
			c.Assert(err, check.IsNil, cmt)
			c.Check(b.Build(), check.DeepEquals, test.ExpectedCLI, cmt)
			if test.ExpectedLog == "" {
				test.ExpectedLog = RedactCLI(test.ExpectedCLI)
			}
			c.Check(fmt.Sprint(b), check.Equals, test.ExpectedLog, cmt)
		} else {
			if errors.Cause(err) != nil {
				c.Assert(errors.Cause(err), check.Equals, test.ExpectedErr, cmt)
			} else {
				c.Assert(err, check.Equals, test.ExpectedErr, cmt)
			}
		}

		if test.Logger != nil {
			log := test.Logger.(*StringLogger)
			cmtLog := check.Commentf("FAIL: %v\nlog %#v expected to match %#v", test.Name, log, test.LoggerRegex)
			for _, regex := range test.LoggerRegex {
				c.Assert(log.MatchString(regex), check.Equals, true, cmtLog)
			}
		}
	}
}

// NewCommandSuite creates a new CommandSuite.
func NewCommandSuite(tests []CommandTest) *CommandSuite {
	return &CommandSuite{Tests: tests}
}
