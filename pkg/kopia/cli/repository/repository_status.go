package repository

import (
	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/command"
	flagcommon "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/common"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/safecli"
)

// StatusArgs defines the arguments for the `kopia repository status ...` command.
type StatusArgs struct {
	cli.CommonArgs

	JSONOutput bool // shows the output in JSON format

	Logger log.Logger
}

// Status creates a new `kopia repository status ...` command.
func Status(args StatusArgs) (safecli.CommandBuilder, error) {
	return command.NewKopiaCommandBuilder(args.CommonArgs,
		command.Repository, command.Status,
		flagcommon.JSONOutput(args.JSONOutput),
	)
}
