package snapshot

import (
	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/command"
	"github.com/kanisterio/kanister/pkg/safecli"

	flagcommon "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/common"
	flagrestore "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/restore"
)

// RestoreArgs defines the arguments for the `kopia restore ...` command.
type RestoreArgs struct {
	cli.CommonArgs
	SnapshotID             string // the snapshot ID to restore
	TargetPath             string // the target path to restore to
	SparseRestore          bool   // write sparse files
	IgnorePermissionErrors bool   // ignore permission errors
}

// Restore creates a new `kopia restore ...` command.
func Restore(args RestoreArgs) (safecli.CommandBuilder, error) {
	return command.NewKopiaCommandBuilder(args.CommonArgs,
		command.Snapshot, command.Restore,
		flagcommon.ID(args.SnapshotID),
		flagrestore.TargetPath(args.TargetPath),
		flagrestore.IgnorePermissionErrors(args.IgnorePermissionErrors),
		flagrestore.WriteSparseFiles(args.SparseRestore),
	)
}
