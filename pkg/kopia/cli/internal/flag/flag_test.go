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

package flag_test

import (
	"errors"
	"testing"

	"gopkg.in/check.v1"

	"github.com/kanisterio/safecli"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
)

var (
	ErrFlag = errors.New("flag error")
)

// MockFlagApplier is a mock implementation of the FlagApplier interface.
type MockFlagApplier struct {
	flagName string
	applyErr error
}

func (m *MockFlagApplier) Apply(cli safecli.CommandAppender) error {
	cli.AppendLoggable(m.flagName)
	return m.applyErr
}

func TestApply(t *testing.T) { check.TestingT(t) }

var _ = check.Suite(&test.FlagSuite{Cmd: "cmd", Tests: []test.FlagTest{
	{
		Name:        "Apply with no flags should generate only the command",
		ExpectedCLI: []string{"cmd"},
	},
	{
		Name:        "Apply with nil flags should generate only the command",
		Flag:        flag.NewFlags(nil, nil),
		ExpectedCLI: []string{"cmd"},
	},
	{
		Name: "Apply with flags should generate the command and flags",
		Flag: flag.NewFlags(
			&MockFlagApplier{flagName: "--flag1", applyErr: nil},
			&MockFlagApplier{flagName: "--flag2", applyErr: nil},
		),
		ExpectedCLI: []string{"cmd", "--flag1", "--flag2"},
	},
	{
		Name: "Apply with one error flag should not modify the command and return the error",
		Flag: flag.NewFlags(
			&MockFlagApplier{flagName: "flag1", applyErr: nil},
			&MockFlagApplier{flagName: "flag2", applyErr: ErrFlag},
		),
		ExpectedCLI: []string{"cmd"},
		ExpectedErr: ErrFlag,
	},
	{
		Name: "NewBoolFlag",
		Flag: flag.NewFlags(
			flag.NewBoolFlag("--flag1", true),
			flag.NewBoolFlag("--flag2", false),
		),
		ExpectedCLI: []string{"cmd", "--flag1"},
	},
	{
		Name: "NewBoolFlag with empty flag name should return an error",
		Flag: flag.NewFlags(
			flag.NewBoolFlag("", true),
		),
		ExpectedCLI: []string{"cmd"},
		ExpectedErr: cli.ErrInvalidFlag,
	},
	{
		Name: "NewStringFlag",
		Flag: flag.NewFlags(
			flag.NewStringFlag("--flag1", "value1"),
			flag.NewStringFlag("--flag2", ""),
		),
		ExpectedCLI: []string{"cmd", "--flag1=value1"},
	},
	{
		Name: "NewStringFlag with all empty values should return an error",
		Flag: flag.NewFlags(
			flag.NewStringFlag("--flag1", "value1"),
			flag.NewStringFlag("", ""),
		),
		ExpectedCLI: []string{"cmd"},
		ExpectedErr: cli.ErrInvalidFlag,
	},
	{
		Name: "NewRedactedStringFlag",
		Flag: flag.NewFlags(
			flag.NewRedactedStringFlag("--flag1", "value1"),
			flag.NewRedactedStringFlag("--flag2", ""),
			flag.NewRedactedStringFlag("", "value3"),
		),
		ExpectedCLI: []string{"cmd", "--flag1=value1", "value3"},
		ExpectedLog: "cmd --flag1=<****> <****>",
	},
	{
		Name: "NewRedactedStringFlag with all empty values should return an error",
		Flag: flag.NewFlags(
			flag.NewRedactedStringFlag("--flag1", "value1"),
			flag.NewRedactedStringFlag("", ""),
		),
		ExpectedCLI: []string{"cmd"},
		ExpectedErr: cli.ErrInvalidFlag,
	},
	{
		Name: "NewStringValue",
		Flag: flag.NewFlags(
			flag.NewStringArgument("value1"),
		),
		ExpectedCLI: []string{"cmd", "value1"},
	},
	{
		Name: "NewStringValue with empty value should return an error",
		Flag: flag.NewFlags(
			flag.NewStringArgument(""),
		),
		ExpectedCLI: []string{"cmd"},
		ExpectedErr: cli.ErrInvalidFlag,
	},
	{
		Name: "NewFlags should generate multiple flags",
		Flag: flag.NewFlags(
			flag.NewStringFlag("--flag1", "value1"),
			flag.NewRedactedStringFlag("--flag2", "value2"),
			flag.NewStringArgument("value3"),
		),
		ExpectedCLI: []string{"cmd", "--flag1=value1", "--flag2=value2", "value3"},
		ExpectedLog: "cmd --flag1=value1 --flag2=<****> value3",
	},
	{
		Name: "NewFlags should generate no flags if one of them returns an error",
		Flag: flag.NewFlags(
			flag.NewStringFlag("--flag1", "value1"),
			&MockFlagApplier{flagName: "flag2", applyErr: ErrFlag},
		),
		ExpectedCLI: []string{"cmd"},
		ExpectedErr: ErrFlag,
	},
	{
		Name:        "EmptyFlag should not generate any flags",
		Flag:        flag.EmptyFlag(),
		ExpectedCLI: []string{"cmd"},
	},
	{
		Name:        "ErrorFlag should return an error",
		Flag:        flag.ErrorFlag(ErrFlag),
		ExpectedCLI: []string{"cmd"},
		ExpectedErr: ErrFlag,
	},
}})
