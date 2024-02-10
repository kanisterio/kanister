package blob

import (
	"github.com/kanisterio/safecli"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/command"
)

// ListArgs defines the arguments for the `kopia blob list` command.
type ListArgs struct {
	cli.CommonArgs
}

// Create creates a new `kopia blob list ...` command.
func List(args ListArgs) (safecli.CommandBuilder, error) {
	return command.NewKopiaCommandBuilder(args.CommonArgs, command.Blob, command.List)
}
