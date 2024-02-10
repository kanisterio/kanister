package snapshot

import (
	"github.com/kanisterio/safecli"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/command"

	flagcommon "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/common"
	flagrestore "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/restore"
)

// DeleteArgs defines the arguments for the `kopia snapshot delete ...` command.
type DeleteArgs struct {
	cli.CommonArgs
	SnapshotID string // the snapshot ID to delete
}

// Delete creates a new `kopia snapshot delete ...` command.
func Delete(args DeleteArgs) (safecli.CommandBuilder, error) {
	return command.NewKopiaCommandBuilder(args.CommonArgs,
		command.Snapshot, command.Delete,
		flagcommon.ID(args.SnapshotID),
		flagrestore.UnsafeIgnoreSource(true),
	)
}
