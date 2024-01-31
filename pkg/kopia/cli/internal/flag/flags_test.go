package flag

import (
	"errors"
	"testing"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/safecli"
	"gopkg.in/check.v1"
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
				NewStringValue("value1"),
			},
			ExpectedCLI: []string{"cmd", "value1"},
		},
		{
			Name: "NewStringValue with empty value should return an error",
			Flags: []Applier{
				NewStringValue(""),
			},
			ExpectedCLI: []string{"cmd"},
			ExpectedErr: cli.ErrInvalidFlag,
		},
		{
			Name: "NewFlags should generate multiple flags",
			Flags: []Applier{NewFlags(
				NewStringFlag("--flag1", "value1"),
				NewRedactedStringFlag("--flag2", "value2"),
				NewStringValue("value3"),
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

func (s *ApplySuite) TestSwitchFlag(c *check.C) {
	sf := SwitchFlag("--flag")
	c.Assert(sf, check.NotNil)
	b := safecli.NewBuilder()
	c.Assert(sf.Apply(b), check.IsNil)
	c.Assert(b.Build(), check.DeepEquals, []string{"--flag"})
}
