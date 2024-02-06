package test_test

import (
	"strings"
	"testing"

	"github.com/pkg/errors"

	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
	"github.com/kanisterio/safecli"
)

func TestCustomFlag(t *testing.T) { check.TestingT(t) }

// CustomFlagTest is a test for FlagTest.
// it has a custom flag that can be used to test the flag.
// and implements flag.Applier.
type CustomFlagTest struct {
	name        string
	flag        string
	flagErr     error
	expectedErr error
}

func (t *CustomFlagTest) Apply(cli safecli.CommandAppender) error {
	if t.flagErr == nil {
		cli.AppendLoggable(t.flag)
	}
	return t.flagErr
}

func (t *CustomFlagTest) Test(c *check.C) {
	flagTest := test.FlagTest{
		Name:        t.name,
		Flag:        t,
		ExpectedErr: t.expectedErr,
	}
	if t.flag != "" {
		flagTest.ExpectedCLI = []string{t.flag}
	}
	b := safecli.NewBuilder()
	flagTest.Test(c, b)
}

type CustomFlagSuite struct {
	cmd   string
	tests []test.FlagTest
}

func (s *CustomFlagSuite) Test(c *check.C) {
	suite := test.NewFlagSuite(s.tests)
	suite.Cmd = s.cmd
	suite.TestFlags(c)
}

// TestRunnerWithConfig is a test suite for CustomFlagTest.
type TestRunnerWithConfig struct {
	out strings.Builder // output buffer for the test results
	cfg *check.RunConf  // custom test configuration
}

// register the test suite
var _ = check.Suite(&TestRunnerWithConfig{})

// SetUpTest sets up the test suite for running.
// it initializes the output buffer and the test configuration.
func (s *TestRunnerWithConfig) SetUpTest(c *check.C) {
	s.out = strings.Builder{}
	s.cfg = &check.RunConf{
		Output:  &s.out,
		Verbose: true,
	}
}

// TestFlagTestOK tests the FlagTest with no errors.
func (s *TestRunnerWithConfig) TestFlagTestOK(c *check.C) {
	cft := CustomFlagTest{
		name: "TestFlagOK",
		flag: "--test",
	}
	res := check.Run(&cft, s.cfg)
	c.Assert(s.out.String(), check.Matches, "PASS: .*CustomFlagTest\\.Test.*\n")
	c.Assert(res.Passed(), check.Equals, true)
}

// TestFlagTestErr tests the FlagTest with an error.
func (s *TestRunnerWithConfig) TestFlagTestErr(c *check.C) {
	err := errors.New("test error")
	cft := CustomFlagTest{
		name:        "TestFlagErr",
		flagErr:     err,
		expectedErr: err,
	}
	res := check.Run(&cft, s.cfg)
	c.Assert(s.out.String(), check.Matches, "PASS: .*CustomFlagTest\\.Test.*\n")
	c.Assert(res.Passed(), check.Equals, true)
}

// TestFlagTestWrapperErr tests the FlagTest with a wrapped error.
func (s *TestRunnerWithConfig) TestFlagTestWrapperErr(c *check.C) {
	err := errors.New("test error")
	werr := errors.Wrap(err, "wrapper error")
	cft := CustomFlagTest{
		name:        "TestFlagTestWrapperErr",
		flagErr:     werr,
		expectedErr: err,
	}
	res := check.Run(&cft, s.cfg)
	c.Assert(s.out.String(), check.Matches, "PASS: .*CustomFlagTest\\.Test.*\n")
	c.Assert(res.Passed(), check.Equals, true)
}

// TestFlagTestUnexpectedErr tests the FlagTest with an unexpected error.
func (s *TestRunnerWithConfig) TestFlagTestUnexpectedErr(c *check.C) {
	err := errors.New("test error")
	cft := CustomFlagTest{
		name:        "TestFlagUnexpectedErr",
		flag:        "--test",
		flagErr:     err,
		expectedErr: nil,
	}
	res := check.Run(&cft, s.cfg)
	ss := s.out.String()
	c.Assert(strings.Contains(ss, "TestFlagUnexpectedErr"), check.Equals, true)
	c.Assert(strings.Contains(ss, "test error"), check.Equals, true)
	c.Assert(res.Passed(), check.Equals, false)
}

// TestFlagSuiteOK tests the FlagSuite with no errors.
func (s *TestRunnerWithConfig) TestFlagSuiteOK(c *check.C) {
	cfs := CustomFlagSuite{
		cmd: "cmd",
		tests: []test.FlagTest{
			{
				Name:        "TestFlagOK",
				Flag:        &CustomFlagTest{name: "TestFlagOK", flag: "--test"},
				ExpectedCLI: []string{"cmd", "--test"},
			},
		},
	}
	res := check.Run(&cfs, s.cfg)
	c.Assert(s.out.String(), check.Matches, "PASS: .*CustomFlagSuite\\.Test.*\n")
	c.Assert(res.Passed(), check.Equals, true)
}
