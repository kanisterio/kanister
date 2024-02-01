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

package flag

import (
	"errors"
	"testing"

	"gopkg.in/check.v1"

	"github.com/kanisterio/safecli"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
)

var (
	ErrFlag = errors.New("flag error")
)

// MockFlagApplier is a mock implementation of the FlagApplier interface.
type MockFlagApplier struct {
	flagName string
	applyErr error
}

func (m *MockFlagApplier) Flag() string {
	return m.flagName
}

func (m *MockFlagApplier) Apply(cli safecli.CommandAppender) error {
	cli.AppendLoggable(m.flagName)
	return m.applyErr
}

func TestApply(t *testing.T) { check.TestingT(t) }

type ApplySuite struct{}

var _ = check.Suite(&ApplySuite{})

func (s *ApplySuite) TestApply(c *check.C) {
	tests := []struct {
		Name        string
		Flags       []Applier
		ExpectedCLI []string
		ExpectedLog string
		ExpectedErr error
	}{
		{
			Name:        "Apply with no flags should generate only the command",
			ExpectedCLI: []string{"cmd"},
		},
		{
			Name:        "Apply with nil flags should generate only the command",
			Flags:       []Applier{nil, nil},
			ExpectedCLI: []string{"cmd"},
		},
		{
			Name: "Apply with flags should generate the command and flags",
			Flags: []Applier{
				&MockFlagApplier{flagName: "--flag1", applyErr: nil},
				&MockFlagApplier{flagName: "--flag2", applyErr: nil},
			},
			ExpectedCLI: []string{"cmd", "--flag1", "--flag2"},
		},
		{
			Name: "Apply with one error flag should not modify the command and return the error",
			Flags: []Applier{
				&MockFlagApplier{flagName: "flag1", applyErr: nil},
				&MockFlagApplier{flagName: "flag2", applyErr: ErrFlag},
			},
			ExpectedCLI: []string{"cmd"},
			ExpectedErr: ErrFlag,
		},
		{
			Name: "NewBoolFlag",
			Flags: []Applier{
				NewBoolFlag("--flag1", true),
				NewBoolFlag("--flag2", false),
			},
			ExpectedCLI: []string{"cmd", "--flag1"},
		},
		{
			Name: "NewBoolFlag with empty flag name should return an error",
			Flags: []Applier{
				NewBoolFlag("", true),
			},
			ExpectedCLI: []string{"cmd"},
			ExpectedErr: cli.ErrInvalidFlag,
		},
		{
			Name: "NewStringFlag",
			Flags: []Applier{
				NewStringFlag("--flag1", "value1"),
				NewStringFlag("--flag2", ""),
			},
			ExpectedCLI: []string{"cmd", "--flag1=value1"},
		},
		{
			Name: "NewStringFlag with all empty values should return an error",
			Flags: []Applier{
				NewStringFlag("--flag1", "value1"),
				NewStringFlag("", ""),
			},
			ExpectedCLI: []string{"cmd"},
			ExpectedErr: cli.ErrInvalidFlag,
		},
		{
			Name: "NewRedactedStringFlag",
			Flags: []Applier{
				NewRedactedStringFlag("--flag1", "value1"),
				NewRedactedStringFlag("--flag2", ""),
				NewRedactedStringFlag("", "value3"),
			},
			ExpectedCLI: []string{"cmd", "--flag1=value1", "value3"},
			ExpectedLog: "cmd --flag1=<****> <****>",
		},
		{
			Name: "NewRedactedStringFlag with all empty values should return an error",
			Flags: []Applier{
				NewRedactedStringFlag("--flag1", "value1"),
				NewRedactedStringFlag("", ""),
			},
			ExpectedCLI: []string{"cmd"},
			ExpectedErr: cli.ErrInvalidFlag,
		},
		{
			Name: "NewStringValue",
			Flags: []Applier{
				NewStringArgument("value1"),
			},
			ExpectedCLI: []string{"cmd", "value1"},
		},
		{
			Name: "NewStringValue with empty value should return an error",
			Flags: []Applier{
				NewStringArgument(""),
			},
			ExpectedCLI: []string{"cmd"},
			ExpectedErr: cli.ErrInvalidFlag,
		},
		{
			Name: "NewFlags should generate multiple flags",
			Flags: []Applier{NewFlags(
				NewStringFlag("--flag1", "value1"),
				NewRedactedStringFlag("--flag2", "value2"),
				NewStringArgument("value3"),
			)},
			ExpectedCLI: []string{"cmd", "--flag1=value1", "--flag2=value2", "value3"},
			ExpectedLog: "cmd --flag1=value1 --flag2=<****> value3",
		},
		{
			Name:        "DoNothingFlag should not generate any flags",
			Flags:       []Applier{DoNothingFlag()},
			ExpectedCLI: []string{"cmd"},
		},
		{
			Name:        "ErrorFlag should return an error",
			Flags:       []Applier{ErrorFlag(ErrFlag)},
			ExpectedCLI: []string{"cmd"},
			ExpectedErr: ErrFlag,
		},
	}

	for _, tt := range tests {
		b := safecli.NewBuilder("cmd")
		err := Apply(b, tt.Flags...)
		cmt := check.Commentf("FAIL: %v", tt.Name)
		if tt.ExpectedErr == nil {
			c.Assert(err, check.IsNil, cmt)
		} else {
			c.Assert(err, check.Equals, tt.ExpectedErr, cmt)
		}
		c.Assert(b.Build(), check.DeepEquals, tt.ExpectedCLI, cmt)
		if tt.ExpectedLog != "" {
			c.Assert(b.String(), check.Equals, tt.ExpectedLog, cmt)
		}
	}
}
