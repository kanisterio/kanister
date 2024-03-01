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
	"github.com/kanisterio/safecli/test"
	"gopkg.in/check.v1"

	intlog "github.com/kanisterio/kanister/pkg/kopia/cli/internal/log"
	"github.com/kanisterio/kanister/pkg/log"
)

// ArgumentTest extends test.ArgumentTest to include logger tests.
type ArgumentTest struct {
	test.ArgumentTest

	Logger      log.Logger // Logger is the logger to use for the test. (optional)
	LoggerRegex []string   // LoggerRegex is a list of regexs to match against the log output. (optional)
}

// Test runs the test with the given command and checks the log output.
func (t *ArgumentTest) Test(c *check.C, cmd string) {
	t.ArgumentTest.Test(c, cmd)
	t.assertLog(c)
}

// assertLog checks the log output against the expected regexs.
func (t *ArgumentTest) assertLog(c *check.C) {
	if t.Logger == nil {
		if len(t.LoggerRegex) > 0 {
			c.Fatalf("t.Logger is nil but t.LoggerRegex is %#v", t.LoggerRegex)
		}
		return
	}

	log, ok := t.Logger.(*intlog.StringLogger)
	if !ok {
		c.Fatalf("t.Logger is not *intlog.StringLogger")
	}
	if t.isEmptyLogExpected() {
		cmtLog := check.Commentf("FAIL: log should be empty but got %#v", log)
		c.Assert(len([]string(*log)), check.Equals, 0, cmtLog)
		return
	}

	// Check each regex.
	for _, regex := range t.LoggerRegex {
		cmtLog := check.Commentf("FAIL: %v\nlog %#v expected to match %#v", t.ArgumentTest.Name, log, regex)
		c.Assert(log.MatchString(regex), check.Equals, true, cmtLog)
	}
}

// isEmptyLogExpected returns true if the test expects an empty log.
func (t *ArgumentTest) isEmptyLogExpected() bool {
	return len(t.LoggerRegex) == 1 && t.LoggerRegex[0] == ""
}

// ArgumentSuite defines a suite of tests for a single ArgumentTest.
type ArgumentSuite struct {
	Cmd       string         // Cmd appends to the safecli.Builder before test if not empty.
	Arguments []ArgumentTest // Tests to run.
}

// TestArguments runs all tests in the suite.
func (s *ArgumentSuite) TestArguments(c *check.C) {
	for _, arg := range s.Arguments {
		arg.Test(c, s.Cmd)
	}
}
