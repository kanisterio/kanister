package maintenance

import (
	"github.com/kanisterio/safecli"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/command"
	flagcommon "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/common"
)

// InfoArgs defines the arguments for the `kopia maintenance info ...` command.
type InfoArgs struct {
	cli.CommonArgs
	JSONOutput bool // shows the output in JSON format
}

// Info creates a new `kopia maintenance info ...` command.
func Info(args InfoArgs) (safecli.CommandBuilder, error) {
	return command.NewKopiaCommandBuilder(args.CommonArgs,
		command.Maintenance, command.Info,
		flagcommon.JSONOutput(args.JSONOutput),
	)
}
