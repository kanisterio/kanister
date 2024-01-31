package azure

import (
	"testing"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
	"gopkg.in/check.v1"
)

func TestStorageAzureFlags(t *testing.T) { check.TestingT(t) }

var _ = check.Suite(test.NewFlagSuite([]test.FlagTest{
	{
		Name: "Empty AzureCountainer should not generate a flag",
		Flag: AzureCountainer(""),
	},
	{
		Name:        "AzureCountainer with value should generate a flag with the given value",
		Flag:        AzureCountainer("container"),
		ExpectedCLI: []string{"--container=container"},
	},
	{
		Name: "Empty Prefix should not generate a flag",
		Flag: Prefix(""),
	},
	{
		Name:        "Prefix with value should generate a flag with the given value",
		Flag:        Prefix("prefix"),
		ExpectedCLI: []string{"--prefix=prefix"},
	},
}))
