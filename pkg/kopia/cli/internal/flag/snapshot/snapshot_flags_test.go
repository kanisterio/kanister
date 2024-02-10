package snapshot

import (
	"testing"
	"time"

	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
)

func TestSnapshotFlags(t *testing.T) { check.TestingT(t) }

var _ = check.Suite(test.NewFlagSuite([]test.FlagTest{
	{
		Name:        "Parallel with value should generate a flag with the given value",
		Flag:        Parallel(1),
		ExpectedCLI: []string{"--parallel=1"},
	},
	{
		Name:        "Empty ProgressUpdateInterval should generate a flag with default value",
		Flag:        ProgressUpdateInterval(0),
		ExpectedCLI: []string{"--progress-update-interval=1h"},
	},
	{
		Name:        "ProgressUpdateInterval with value should generate a flag with the given interval",
		Flag:        ProgressUpdateInterval(42 * time.Hour),
		ExpectedCLI: []string{"--progress-update-interval=42h"},
	},
	{
		Name:        "Empty PathToBackup should generate an ErrInvalidPathToBackup error",
		Flag:        PathToBackup(""),
		ExpectedErr: cli.ErrInvalidBackupPath,
	},
	{
		Name:        "PathToBackup with value should generate an argument with the given path",
		Flag:        PathToBackup("/foo/bar"),
		ExpectedCLI: []string{"/foo/bar"},
	},
	{
		Name: "Empty Tags should not generate a flag",
		Flag: Tags(nil),
	},
	{
		Name: "Tags with value should generate multiple tag flags with the given values",
		Flag: Tags([]string{
			"tag1:value1",
			"tag2:value2",
		}),
		ExpectedCLI: []string{"--tags=tag1:value1", "--tags=tag2:value2"},
	},
	{
		Name: "Tags with invalid value should generate an ErrInvalidTag error",
		Flag: Tags([]string{
			"tag1",
		}),
		ExpectedErr: cli.ErrInvalidTag,
	},
	{
		Name: "Tags with empty value should generate an ErrInvalidTag error",
		Flag: Tags([]string{
			"",
		}),
		ExpectedErr: cli.ErrInvalidTag,
	},
	{
		Name: "TagsWithNoValidation should ignore validation and generate a flag with the given values",
		Flag: TagsWithNoValidation([]string{
			"tag1",
		}),

		ExpectedCLI: []string{"--tags=tag1"},
	},
}))
