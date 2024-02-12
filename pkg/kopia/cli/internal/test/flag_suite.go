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

// CheckCommentString implements check.CommentInterface
func (t *FlagTest) CheckCommentString() string {
	return t.Name
}

// setDefaultExpectedLog sets the default value for ExpectedLog based on ExpectedCLI.
func (t *FlagTest) setDefaultExpectedLog() {
	if len(t.ExpectedLog) == 0 && len(t.ExpectedCLI) > 0 {
		t.ExpectedLog = strings.Join(t.ExpectedCLI, " ")
	}
}

// assertError checks the error against ExpectedErr.
func (t *FlagTest) assertError(c *check.C, err error) {
	if actualErr := errors.Cause(err); actualErr != nil {
		c.Assert(actualErr, check.Equals, t.ExpectedErr, t)
	} else {
		c.Assert(err, check.Equals, t.ExpectedErr, t)
	}
}

// assertNoError makes sure there is no error.
func (t *FlagTest) assertNoError(c *check.C, err error) {
	c.Assert(err, check.IsNil, t)
}

// assertCLI asserts the builder's CLI output against ExpectedCLI.
func (t *FlagTest) assertCLI(c *check.C, b *safecli.Builder) {
	if t.ExpectedCLI != nil {
		c.Check(b.Build(), check.DeepEquals, t.ExpectedCLI, t)
	}
}

// assertLog asserts the builder's log output against ExpectedLog.
func (t *FlagTest) assertLog(c *check.C, b *safecli.Builder) {
	t.setDefaultExpectedLog()
	c.Check(b.String(), check.Equals, t.ExpectedLog, t)
}

// Test runs the flag test.
func (ft *FlagTest) Test(c *check.C, b *safecli.Builder) {
	err := flag.Apply(b, ft.Flag)
	ft.assertCLI(c, b)
	if ft.ExpectedErr != nil {
		ft.assertError(c, err)
	} else {
		ft.assertNoError(c, err)
		ft.assertLog(c, b)
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
