package blob

import (
	"testing"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
	"gopkg.in/check.v1"
)

func TestBlobFlags(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(test.NewFlagSuite([]test.FlagTest{
	{
		Name:        "Raw should always generate '--raw' flag",
		Flag:        Raw,
		ExpectedCLI: []string{"--raw"},
		ExpectedLog: "--raw",
	},
}))
