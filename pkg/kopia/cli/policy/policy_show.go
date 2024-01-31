package policy

import (
	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/command"
	flagcommon "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/common"
	flagpolicy "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/policy"
	"github.com/kanisterio/kanister/pkg/safecli"
)

// ShowArgs defines the arguments for the `kopia policy show ...` command.
type ShowArgs struct {
	cli.CommonArgs
	JSONOutput bool // shows the output in JSON format
}

// Show creates a new `kopia policy show ...` command.
func Show(args ShowArgs) (safecli.CommandBuilder, error) {
	return command.NewKopiaCommandBuilder(args.CommonArgs,
		command.Policy, command.Show,
		flagpolicy.Global(true),
		flagcommon.JSONOutput(args.JSONOutput),
	)
}
