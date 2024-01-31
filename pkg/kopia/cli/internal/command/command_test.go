package command

import (
	"errors"
	"testing"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/safecli"
	"gopkg.in/check.v1"
)

func TestCommand(t *testing.T) { check.TestingT(t) }

type CommandSuite struct{}

var _ = check.Suite(&CommandSuite{})

type mockCommandAndFlag struct {
	flagName string
	err      error
}

func (m *mockCommandAndFlag) Apply(cli safecli.CommandAppender) error {
	if m.err != nil {
		return m.err
	}
	cli.AppendLoggable(m.flagName)
	return nil
}

func (s *CommandSuite) TestCommand(c *check.C) {
	b := safecli.NewBuilder()
	cmd := Command("cmd")
	err := cmd.Apply(b)
	c.Assert(err, check.IsNil)
	c.Check(b.Build(), check.DeepEquals, []string{"cmd"})
}

func (s *CommandSuite) TestNewKopiaCommandBuilder(c *check.C) {
	b, err := NewKopiaCommandBuilder(cli.CommonArgs{}, &mockCommandAndFlag{flagName: "--flag1"}, &mockCommandAndFlag{flagName: "--flag2"})
	c.Assert(err, check.IsNil)
	c.Check(b.Build(), check.DeepEquals, []string{
		"kopia",
		"--log-level=error",
		"--flag1",
		"--flag2",
	})
}

var (
	errInvalidCommand = errors.New("invalid command")
	errInvalidFlag    = errors.New("invalid flag")
)

func (s *CommandSuite) TestNewCommandBuilderWithFailedCommand(c *check.C) {
	// test if command is invalid
	b, err := NewCommandBuilder(
		&mockCommandAndFlag{err: errInvalidCommand},
	)
	c.Assert(b, check.IsNil)
	c.Assert(err, check.Equals, errInvalidCommand)
}

func (s *CommandSuite) TestNewCommandBuilderWithFailedFlag(c *check.C) {
	// test if flag is invalid
	b, err := NewCommandBuilder(
		&mockCommandAndFlag{flagName: "cmd"},
		&mockCommandAndFlag{err: errInvalidFlag},
	)
	c.Assert(b, check.IsNil)
	c.Assert(err, check.Equals, errInvalidFlag)
}

func (s *CommandSuite) TestNewCommandBuilder(c *check.C) {
	// test if command and flag are valid
	b, err := NewCommandBuilder(
		&mockCommandAndFlag{flagName: "cmd"},
		&mockCommandAndFlag{flagName: "--flag1"},
		&mockCommandAndFlag{flagName: "--flag2"},
	)
	c.Assert(err, check.IsNil)
	c.Check(b.Build(), check.DeepEquals, []string{
		"cmd",
		"--flag1",
		"--flag2",
	})
}
