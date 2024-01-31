package maintenance

import (
	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/command"
	"github.com/kanisterio/kanister/pkg/safecli"
)

// RunArgs defines the arguments for the `kopia maintenance run ...` command.
type RunArgs struct {
	cli.CommonArgs
}

// Info creates a new `kopia maintenance run ...` command.
func Run(args RunArgs) (safecli.CommandBuilder, error) {
	return command.NewKopiaCommandBuilder(args.CommonArgs,
		command.Maintenance, command.Run,
	)
}
