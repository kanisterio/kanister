package gcs

import (
	"testing"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
	"gopkg.in/check.v1"
)

func TestStorageGCSFlags(t *testing.T) { check.TestingT(t) }

var _ = check.Suite(test.NewFlagSuite([]test.FlagTest{
	{
		Name: "Empty CredentialsFile should not generate a flag",
		Flag: CredentialsFile(""),
	},
	{
		Name:        "CredentialsFile with value should generate a flag with the given value",
		Flag:        CredentialsFile("/path/to/credentials"),
		ExpectedCLI: []string{"--credentials-file=/path/to/credentials"},
	},
}))
