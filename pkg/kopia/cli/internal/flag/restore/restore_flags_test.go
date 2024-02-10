package restore

import (
	"testing"

	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
)

func TestRepositoryFlags(t *testing.T) { check.TestingT(t) }

var _ = check.Suite(test.NewFlagSuite([]test.FlagTest{
	{
		Name:        "Empty TargetPath should fail with ErrInvalidTargetPath error",
		Flag:        TargetPath(""),
		ExpectedErr: cli.ErrInvalidTargetPath,
	},
	{
		Name:        "TargetPath with value should generate a flag with given value",
		Flag:        TargetPath("/target/path"),
		ExpectedCLI: []string{"/target/path"},
	},
	{
		Name:        "IgnorePermissionErrors(false) should generate --no-ignore-permission-errors flag",
		Flag:        IgnorePermissionErrors(false),
		ExpectedCLI: []string{"--no-ignore-permission-errors"},
	},
	{
		Name:        "IgnorePermissionErrors(true) should generate --ignore-permission-errors flag",
		Flag:        IgnorePermissionErrors(true),
		ExpectedCLI: []string{"--ignore-permission-errors"},
	},
	{
		Name: "WriteSparseFiles(false) should not generate a flag",
		Flag: WriteSparseFiles(false),
	},
	{
		Name:        "WriteSparseFiles(true)should generate a flag",
		Flag:        WriteSparseFiles(true),
		ExpectedCLI: []string{"--write-sparse-files"},
	},
	{
		Name: "UnsafeIgnoreSource(false) should not generate a flag",
		Flag: UnsafeIgnoreSource(false),
	},
	{
		Name:        "UnsafeIgnoreSource(true) should generate a flag",
		Flag:        UnsafeIgnoreSource(true),
		ExpectedCLI: []string{"--unsafe-ignore-source"},
	},
}))
