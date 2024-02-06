package test

import (
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

// CheckCommentString implements check.CommentInterface
func (t *FlagTest) CheckCommentString() string {
	return t.Name
}

// setDefaultExpectedLog sets the default value for ExpectedLog based on ExpectedCLI.
func (t *FlagTest) setDefaultExpectedLog() {
	if len(t.ExpectedLog) == 0 && len(t.ExpectedCLI) > 0 {
		t.ExpectedLog = RedactCLI(t.ExpectedCLI)
	}
}

// assertError checks the error against ExpectedErr.
func (t *FlagTest) assertError(c *check.C, err error) {
	actualErr := errors.Cause(err)
	c.Assert(actualErr, check.Equals, t.ExpectedErr, t)
}

// assertNoError makes sure there is no error.
func (t *FlagTest) assertNoError(c *check.C, err error) {
	c.Assert(err, check.IsNil, t)
}

// assertCLI asserts the builder's CLI output against ExpectedCLI.
func (t *FlagTest) assertCLI(c *check.C, b *safecli.Builder) {
	c.Check(b.Build(), check.DeepEquals, t.ExpectedCLI, t)
}

// assertLog asserts the builder's log output against ExpectedLog.
func (t *FlagTest) assertLog(c *check.C, b *safecli.Builder) {
	t.setDefaultExpectedLog()
	c.Check(b.String(), check.Equals, t.ExpectedLog, t)
}

// Test runs the flag test.
func (t *FlagTest) Test(c *check.C, b *safecli.Builder) {
	err := flag.Apply(b, t.Flag)
	if t.ExpectedErr != nil {
		t.assertError(c, err)
	} else {
		t.assertNoError(c, err)
		t.assertCLI(c, b)
		t.assertLog(c, b)
	}
}

// FlagSuite defines a test suite for flags.
type FlagSuite struct {
	Cmd   string     // Cmd appends to the safecli.Builder before test if not empty.
	Tests []FlagTest // Tests to run.
}

// TestFlags runs all tests in the flag suite.
func (s *FlagSuite) TestFlags(c *check.C) {
	for _, test := range s.Tests {
		b := newBuilder(s.Cmd)
		test.Test(c, b)
	}
}

// NewFlagSuite creates a new FlagSuite.
func NewFlagSuite(tests []FlagTest) *FlagSuite {
	return &FlagSuite{Tests: tests}
}

// newBuilder creates a new safecli.Builder with the given command.
func newBuilder(cmd string) *safecli.Builder {
	builder := safecli.NewBuilder()
	if cmd != "" {
		builder.AppendLoggable(cmd)
	}
	return builder
}
