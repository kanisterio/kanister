package blob

import (
	"testing"

	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
)

func TestBlobFlags(t *testing.T) { check.TestingT(t) }

var _ = check.Suite(test.NewFlagSuite([]test.FlagTest{
	{
		Name:        "Raw should always generate '--raw' flag",
		Flag:        Raw,
		ExpectedCLI: []string{"--raw"},
	},
}))
