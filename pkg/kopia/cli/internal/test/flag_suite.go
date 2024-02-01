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
	"strings"

	"gopkg.in/check.v1"

	"github.com/pkg/errors"

	"github.com/kanisterio/safecli"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag"
)

// FlagTest defines a single test for a flag.
type FlagTest struct {
	// Name of the test. (required)
	Name string

	// Flag to test. (required)
	Flag flag.Applier

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
}

// FlagSuite defines a test suite for flags.
type FlagSuite struct {
	Tests []FlagTest
}

// TestFlags runs all tests in the suite.
func (s *FlagSuite) TestFlags(c *check.C) {
	for _, test := range s.Tests {
		b := safecli.NewBuilder()
		err := test.Flag.Apply(b)
		cmt := check.Commentf("FAIL: '%v'", test.Name)
		if test.ExpectedErr == nil {
			c.Assert(err, check.IsNil, cmt)
			c.Check(b.Build(), check.DeepEquals, test.ExpectedCLI, cmt)
			if test.ExpectedLog == "" {
				test.ExpectedLog = strings.Join(test.ExpectedCLI, " ")
			}
			c.Check(b.String(), check.Equals, test.ExpectedLog, cmt)
		} else {
			if errors.Cause(err) != nil {
				c.Assert(errors.Cause(err), check.Equals, test.ExpectedErr, cmt)
			} else {
				c.Assert(err, check.Equals, test.ExpectedErr, cmt)
			}
		}
	}
}

// NewFlagSuite creates a new FlagSuite.
func NewFlagSuite(tests []FlagTest) *FlagSuite {
	return &FlagSuite{Tests: tests}
}
