package manifest

import (
	"testing"

	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
)

func TestManifestFlags(t *testing.T) { check.TestingT(t) }

var _ = check.Suite(test.NewFlagSuite([]test.FlagTest{
	{
		Name: "Empty Filter should not generate a flag",
		Flag: Filter(""),
	},
	{
		Name:        "Filter with value should generate a flag with given value",
		Flag:        Filter("filter"),
		ExpectedCLI: []string{"--filter=filter"},
	},
}))
