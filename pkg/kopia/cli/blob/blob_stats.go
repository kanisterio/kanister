package blob

import (
	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/command"
	flagblob "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/blob"
	"github.com/kanisterio/kanister/pkg/safecli"
)

// StatsArgs defines the arguments for the `kopia blob stats` command.
type StatsArgs struct {
	cli.CommonArgs
}

// Create creates a new `kopia blob stats ...` command.
func Stats(args StatsArgs) (safecli.CommandBuilder, error) {
	return command.NewKopiaCommandBuilder(args.CommonArgs,
		command.Blob, command.Stats,
		flagblob.Raw,
	)
}
