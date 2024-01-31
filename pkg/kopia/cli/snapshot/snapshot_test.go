package snapshot

import (
	"testing"
	"time"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
	"github.com/kanisterio/kanister/pkg/safecli"
	"gopkg.in/check.v1"
)

func TestSnapshotCommands(t *testing.T) { check.TestingT(t) }

// Test Snapshot commands
var _ = check.Suite(test.NewCommandSuite([]test.CommandTest{
	{
		Name: "SnapshotCreate with default ProgressUpdateInterval",
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
			"--log-level=info",
			"--config-file=path/kopia.config",
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
		Name: "SnapshotCreate with custom ProgressUpdateInterval",
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
			"--log-level=info",
			"--config-file=path/kopia.config",
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
		Name: "SnapshotExpire",
		CLI: func() (safecli.CommandBuilder, error) {
			args := ExpireArgs{
				CommonArgs: test.CommonArgs,
				RootID:     "root-id",
				MustDelete: true,
			}
			return Expire(args)
		},
		ExpectedCLI: []string{"kopia",
			"--log-level=error",
			"--config-file=path/kopia.config",
			"--log-dir=cache/log",
			"--password=encr-key",
			"snapshot",
			"expire",
			"root-id",
			"--delete",
		},
	},
	{
		Name: "SnapshotExpire without delete",
		CLI: func() (safecli.CommandBuilder, error) {
			args := ExpireArgs{
				CommonArgs: test.CommonArgs,
				RootID:     "root-id",
			}
			return Expire(args)
		},
		ExpectedCLI: []string{"kopia",
			"--log-level=error",
			"--config-file=path/kopia.config",
			"--log-dir=cache/log",
			"--password=encr-key",
			"snapshot",
			"expire",
			"root-id",
		},
	},
	{
		Name: "SnapshotRestore",
		CLI: func() (safecli.CommandBuilder, error) {
			args := RestoreArgs{
				CommonArgs: test.CommonArgs,
				SnapshotID: "snapshot-id",
				TargetPath: "target/path",
			}
			return Restore(args)
		},
		ExpectedCLI: []string{"kopia",
			"--log-level=error",
			"--config-file=path/kopia.config",
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
		Name: "SnapshotRestore with SparseRestore and IgnorePermissionErrors",
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
			"--log-level=error",
			"--config-file=path/kopia.config",
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
		Name: "SnapshotDelete",
		CLI: func() (safecli.CommandBuilder, error) {
			args := DeleteArgs{
				CommonArgs: test.CommonArgs,
				SnapshotID: "snapshot-id",
			}
			return Delete(args)
		},
		ExpectedCLI: []string{"kopia",
			"--log-level=error",
			"--config-file=path/kopia.config",
			"--log-dir=cache/log",
			"--password=encr-key",
			"snapshot",
			"delete",
			"snapshot-id",
			"--unsafe-ignore-source",
		},
	},
	{
		Name: "SnapshotList",
		CLI: func() (safecli.CommandBuilder, error) {
			args := ListArgs{
				CommonArgs: test.CommonArgs,
			}
			return List(args)
		},
		ExpectedCLI: []string{"kopia",
			"--log-level=error",
			"--config-file=path/kopia.config",
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
		Name: "SnapshotList with Tags",
		CLI: func() (safecli.CommandBuilder, error) {
			args := ListArgs{
				CommonArgs:   test.CommonArgs,
				FilterByTags: []string{"tag1:val1", "tag2:val2"},
			}
			return List(args)
		},
		ExpectedCLI: []string{"kopia",
			"--log-level=error",
			"--config-file=path/kopia.config",
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
