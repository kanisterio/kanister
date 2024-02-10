package restore

import (
	"github.com/kanisterio/safecli"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/command"
	flagcommon "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/common"
	flagrestore "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/restore"
)

// RestoreArgs defines the arguments for the `kopia restore ...` command.
type RestoreArgs struct {
	cli.CommonArgs
	RootID                 string // the root entry to restore
	TargetPath             string // the target path to restore to
	IgnorePermissionErrors bool   // ignore permission errors
}

// Restore creates a new `kopia restore ...` command.
func Restore(args RestoreArgs) (safecli.CommandBuilder, error) {
	return command.NewKopiaCommandBuilder(args.CommonArgs,
		command.Restore,
		flagcommon.ID(args.RootID),
		flagrestore.TargetPath(args.TargetPath),
		flagrestore.IgnorePermissionErrors(args.IgnorePermissionErrors),
	)
}
