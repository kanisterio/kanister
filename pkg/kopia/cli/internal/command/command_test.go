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

package command

import (
	"errors"
	"testing"

	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/safecli"
)

func TestCommand(t *testing.T) { check.TestingT(t) }

type CommandSuite struct{}

var _ = check.Suite(&CommandSuite{})

var (
	errInvalidCommand = errors.New("invalid command")
	errInvalidFlag    = errors.New("invalid flag")
)

type mockCommandAndFlag struct {
	flagName string
	err      error
}

func (m *mockCommandAndFlag) Apply(cli safecli.CommandAppender) error {
	if m.err == nil {
		cli.AppendLoggable(m.flagName)
	}
	return m.err
}

func (s *CommandSuite) TestCommand(c *check.C) {
	b := safecli.NewBuilder()
	cmd := Command{"cmd"}
	err := cmd.Apply(b)
	c.Assert(err, check.IsNil)
	c.Check(b.Build(), check.DeepEquals, []string{"cmd"})
}

func (s *CommandSuite) TestEmptyCommand(c *check.C) {
	b := safecli.NewBuilder()
	cmd := Command{}
	err := cmd.Apply(b)
	c.Assert(err, check.Equals, cli.ErrInvalidCommand)
}

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
