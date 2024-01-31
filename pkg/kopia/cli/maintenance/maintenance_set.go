package maintenance

import (
	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/command"
	flagmaintenance "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/maintenance"
	"github.com/kanisterio/kanister/pkg/safecli"
)

// SetOwnerArgs defines the arguments for the `kopia maintenance set ...` command.
type SetOwnerArgs struct {
	cli.CommonArgs
	CustomOwner string // the owner to set on the repository
}

// SetOwner creates a new `kopia maintenance set ...` command.
func SetOwner(args SetOwnerArgs) (safecli.CommandBuilder, error) {
	return command.NewKopiaCommandBuilder(args.CommonArgs,
		command.Maintenance, command.Set,
		flagmaintenance.CustomerOwner(args.CustomOwner),
	)
}
