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

// CommonArgs is a set of common arguments for the tests.
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
	// if empty, it will be derived from ExpectedCLI by redacting sensitive information.
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

// CheckCommentString implements check.CommentInterface
func (t *CommandTest) CheckCommentString() string {
	return t.Name
}

func (t *CommandTest) setDefaultExpectedLog() {
	if len(t.ExpectedLog) == 0 && len(t.ExpectedCLI) > 0 {
		t.ExpectedLog = RedactCLI(t.ExpectedCLI)
	}
}

func (t *CommandTest) assertError(c *check.C, err error) {
	actualErr := errors.Cause(err)
	c.Assert(actualErr, check.Equals, t.ExpectedErr, t)
}

func (t *CommandTest) assertNoError(c *check.C, err error) {
	c.Assert(err, check.IsNil, t)
}

func (t *CommandTest) assertCLI(c *check.C, b safecli.CommandBuilder) {
	c.Check(b.Build(), check.DeepEquals, t.ExpectedCLI, t)
}

func (t *CommandTest) assertLog(c *check.C, b safecli.CommandBuilder) {
	t.setDefaultExpectedLog()
	c.Check(fmt.Sprint(b), check.Equals, t.ExpectedLog, t)
}

func (t *CommandTest) assertLogger(c *check.C) {
	log, ok := t.Logger.(*StringLogger)
	if !ok {
		c.Fatalf("t.Logger is not a StringLogger")
	}
	cmtLog := check.Commentf("FAIL: %v\nlog %#v expected to match %#v", t.Name, log, t.LoggerRegex)
	for _, regex := range t.LoggerRegex {
		c.Assert(log.MatchString(regex), check.Equals, true, cmtLog)
	}
}

// Test runs the command test.
func (t *CommandTest) Test(c *check.C) {
	b, err := t.CLI()
	if t.ExpectedErr == nil {
		t.assertNoError(c, err)
		t.assertCLI(c, b)
		t.assertLog(c, b)
	} else {
		t.assertError(c, err)
	}
	if t.Logger != nil {
		t.assertLogger(c)
	}
}

// CommandSuite defines a test suite for commands.
type CommandSuite struct {
	Tests []CommandTest
}

// TestCommands runs all tests in the suite.
func (s *CommandSuite) TestCommands(c *check.C) {
	for _, test := range s.Tests {
		test.Test(c)
	}
}

// NewCommandSuite creates a new CommandSuite.
func NewCommandSuite(tests []CommandTest) *CommandSuite {
	return &CommandSuite{Tests: tests}
}
