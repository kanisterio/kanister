package test

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag"
	"github.com/kanisterio/kanister/pkg/safecli"
	"gopkg.in/check.v1"
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
