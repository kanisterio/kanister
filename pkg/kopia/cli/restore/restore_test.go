package restore

import (
	"testing"

	"gopkg.in/check.v1"

	"github.com/kanisterio/safecli"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
)

func TestRestoreCommands(t *testing.T) { check.TestingT(t) }

var _ = check.Suite(test.NewCommandSuite([]test.CommandTest{
	{
		Name: "Empty",
		CLI: func() (safecli.CommandBuilder, error) {
			var args RestoreArgs
			return Restore(args)
		},
		ExpectedErr: cli.ErrInvalidID,
	},
	{
		Name: "Empty TargetPath",
		CLI: func() (safecli.CommandBuilder, error) {
			args := RestoreArgs{
				RootID: "snapshot-id",
			}
			return Restore(args)
		},
		ExpectedErr: cli.ErrInvalidTargetPath,
	},
	{
		Name: "Restore with no-ignore-permission-errors flag",
		CLI: func() (safecli.CommandBuilder, error) {
			args := RestoreArgs{
				CommonArgs: cli.CommonArgs{
					RepoPassword:   "encr-key",
					ConfigFilePath: "path/kopia.config",
					LogDirectory:   "cache/log",
				},
				RootID:     "snapshot-id",
				TargetPath: "target/path",
			}
			return Restore(args)
		},
		ExpectedCLI: []string{
			"kopia",
			"--config-file=path/kopia.config",
			"--log-level=error",
			"--log-dir=cache/log",
			"--password=encr-key",
			"restore",
			"snapshot-id",
			"target/path",
			"--no-ignore-permission-errors",
		},
	},
	{
		Name: "Restore with ignore-permission-errors flag",
		CLI: func() (safecli.CommandBuilder, error) {
			args := RestoreArgs{
				CommonArgs: cli.CommonArgs{
					RepoPassword:   "encr-key",
					ConfigFilePath: "path/kopia.config",
					LogDirectory:   "cache/log",
				},
				RootID:                 "snapshot-id",
				TargetPath:             "target/path",
				IgnorePermissionErrors: true,
			}
			return Restore(args)
		},
		ExpectedCLI: []string{
			"kopia",
			"--config-file=path/kopia.config",
			"--log-level=error",
			"--log-dir=cache/log",
			"--password=encr-key",
			"restore",
			"snapshot-id",
			"target/path",
			"--ignore-permission-errors",
		},
	},
}))
