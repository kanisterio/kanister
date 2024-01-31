package fs

import (
	"testing"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
	"gopkg.in/check.v1"
)

func TestFilestoreFlags(t *testing.T) { check.TestingT(t) }

var _ = check.Suite(test.NewFlagSuite([]test.FlagTest{
	{
		Name: "Empty Path should not generate a flag",
		Flag: Path(""),
	},
	{
		Name:        "Path with value should generate a flag with the given value",
		Flag:        Path("/path/to/file"),
		ExpectedCLI: []string{"--path=/path/to/file"},
	},
}))
