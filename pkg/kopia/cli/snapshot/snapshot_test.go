package snapshot

import (
	"testing"
	"time"

	"github.com/kanisterio/safecli"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
	"gopkg.in/check.v1"
)

func TestSnapshotCommands(t *testing.T) { check.TestingT(t) }

// Test Snapshot commands
var _ = check.Suite(test.NewCommandSuite([]test.CommandTest{
	{
		Name: "snapshot create with default ProgressUpdateInterval",
		CLI: func() (safecli.CommandBuilder, error) {
			args := CreateArgs{
				CommonArgs:             test.CommonArgs,
				PathToBackup:           "path/to/backup",
				ProgressUpdateInterval: 0,
				Parallelism:            8,
			}
			return Create(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-level=info",
			"--log-dir=cache/log",
			"--password=encr-key",
			"snapshot",
			"create",
			"--json",
			"path/to/backup",
			"--parallel=8",
			"--progress-update-interval=1h",
		},
	},
	{
		Name: "snapshot create with custom ProgressUpdateInterval",
		CLI: func() (safecli.CommandBuilder, error) {
			args := CreateArgs{
				CommonArgs:             test.CommonArgs,
				PathToBackup:           "path/to/backup",
				ProgressUpdateInterval: 1*time.Minute + 35*time.Second,
				Parallelism:            8,
			}
			return Create(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-level=info",
			"--log-dir=cache/log",
			"--password=encr-key",
			"snapshot",
			"create",
			"--json",
			"path/to/backup",
			"--parallel=8",
			"--progress-update-interval=2m",
		},
	},
	{
		Name: "snapshot expire",
		CLI: func() (safecli.CommandBuilder, error) {
			args := ExpireArgs{
				CommonArgs: test.CommonArgs,
				RootID:     "root-id",
				MustDelete: true,
			}
			return Expire(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-level=error",
			"--log-dir=cache/log",
			"--password=encr-key",
			"snapshot",
			"expire",
			"root-id",
			"--delete",
		},
	},
	{
		Name: "snapshot expire without delete",
		CLI: func() (safecli.CommandBuilder, error) {
			args := ExpireArgs{
				CommonArgs: test.CommonArgs,
				RootID:     "root-id",
			}
			return Expire(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-level=error",
			"--log-dir=cache/log",
			"--password=encr-key",
			"snapshot",
			"expire",
			"root-id",
		},
	},
	{
		Name: "snapshot restore",
		CLI: func() (safecli.CommandBuilder, error) {
			args := RestoreArgs{
				CommonArgs: test.CommonArgs,
				SnapshotID: "snapshot-id",
				TargetPath: "target/path",
			}
			return Restore(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-level=error",
			"--log-dir=cache/log",
			"--password=encr-key",
			"snapshot",
			"restore",
			"snapshot-id",
			"target/path",
			"--no-ignore-permission-errors",
		},
	},
	{
		Name: "snapshot restore with SparseRestore and IgnorePermissionErrors",
		CLI: func() (safecli.CommandBuilder, error) {
			args := RestoreArgs{
				CommonArgs:             test.CommonArgs,
				SnapshotID:             "snapshot-id",
				TargetPath:             "target/path",
				SparseRestore:          true,
				IgnorePermissionErrors: true,
			}
			return Restore(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-level=error",
			"--log-dir=cache/log",
			"--password=encr-key",
			"snapshot",
			"restore",
			"snapshot-id",
			"target/path",
			"--ignore-permission-errors",
			"--write-sparse-files",
		},
	},
	{
		Name: "snapshot delete",
		CLI: func() (safecli.CommandBuilder, error) {
			args := DeleteArgs{
				CommonArgs: test.CommonArgs,
				SnapshotID: "snapshot-id",
			}
			return Delete(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-level=error",
			"--log-dir=cache/log",
			"--password=encr-key",
			"snapshot",
			"delete",
			"snapshot-id",
			"--unsafe-ignore-source",
		},
	},
	{
		Name: "snapshot list",
		CLI: func() (safecli.CommandBuilder, error) {
			args := ListArgs{
				CommonArgs: test.CommonArgs,
			}
			return List(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-level=error",
			"--log-dir=cache/log",
			"--password=encr-key",
			"snapshot",
			"list",
			"--all",
			"--delta",
			"--show-identical",
			"--json",
		},
	},
	{
		Name: "snapshot list with Tags",
		CLI: func() (safecli.CommandBuilder, error) {
			args := ListArgs{
				CommonArgs:   test.CommonArgs,
				FilterByTags: []string{"tag1:val1", "tag2:val2"},
			}
			return List(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-level=error",
			"--log-dir=cache/log",
			"--password=encr-key",
			"snapshot",
			"list",
			"--all",
			"--delta",
			"--show-identical",
			"--json",
			"--tags=tag1:val1",
			"--tags=tag2:val2",
		},
	},
}))
