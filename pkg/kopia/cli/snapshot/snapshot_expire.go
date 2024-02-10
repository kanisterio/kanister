package snapshot

import (
	"github.com/kanisterio/safecli"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/command"

	flagcommon "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/common"
)

// ExpireArgs defines the arguments for the `kopia snapshot expire ...` command.
type ExpireArgs struct {
	cli.CommonArgs
	RootID     string // the root entry to expire
	MustDelete bool   // delete the expired snapshots
}

// Expire creates a new `kopia snapshot expire ...` command.
func Expire(args ExpireArgs) (safecli.CommandBuilder, error) {
	return command.NewKopiaCommandBuilder(args.CommonArgs,
		command.Snapshot, command.Expire,
		flagcommon.ID(args.RootID),
		flagcommon.Delete(args.MustDelete),
	)
}
