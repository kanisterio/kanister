package test

import (
	"github.com/kanisterio/safecli"
	"github.com/kanisterio/safecli/test"
	"github.com/pkg/errors"
	"gopkg.in/check.v1"
)

// CommandTest defines a single test for a command.
type CommandTest struct {
	// Name of the test. (required)
	Name string

	// Command to test. (required)
	Command func() (*safecli.Builder, error)

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
func (t *CommandTest) CheckCommentString() string {
	return t.Name
}

// setDefaultExpectedLog sets the default value for ExpectedLog based on ExpectedCLI.
func (t *CommandTest) setDefaultExpectedLog() {
	if len(t.ExpectedLog) == 0 && len(t.ExpectedCLI) > 0 {
		t.ExpectedLog = test.RedactCLI(t.ExpectedCLI)
	}
}

// assertNoError makes sure there is no error.
func (t *CommandTest) assertNoError(c *check.C, err error) {
	c.Assert(err, check.IsNil, t)
}

// assertError checks the error against ExpectedErr.
func (t *CommandTest) assertError(c *check.C, err error) {
	actualErr := errors.Cause(err)
	c.Assert(actualErr, check.Equals, t.ExpectedErr, t)
}

// assertCLI asserts the builder's CLI output against ExpectedCLI.
func (t *CommandTest) assertCLI(c *check.C, b *safecli.Builder) {
	if t.ExpectedCLI != nil {
		c.Check(b.Build(), check.DeepEquals, t.ExpectedCLI, t)
	}
}

// assertLog asserts the builder's log output against ExpectedLog.
func (t *CommandTest) assertLog(c *check.C, b *safecli.Builder) {
	if t.ExpectedCLI != nil {
		t.setDefaultExpectedLog()
		c.Check(b.String(), check.Equals, t.ExpectedLog, t)
	}
}

func (t *CommandTest) Test(c *check.C) {
	cmd, err := t.Command()
	if t.ExpectedErr == nil {
		t.assertNoError(c, err)
	} else {
		t.assertError(c, err)
	}
	t.assertCLI(c, cmd)
	t.assertLog(c, cmd)
}

// CommandSuite defines a test suite for commands.
type CommandSuite struct {
	Commands []CommandTest
}

// TestCommands runs all tests in the suite.
func (s *CommandSuite) TestCommands(c *check.C) {
	for _, cmd := range s.Commands {
		cmd.Test(c)
	}
}

// NewCommandSuite creates a new CommandSuite.
func NewCommandSuite(commands []CommandTest) *CommandSuite {
	return &CommandSuite{Commands: commands}
}
