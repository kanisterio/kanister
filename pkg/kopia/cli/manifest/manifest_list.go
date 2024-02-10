package manifest

import (
	"github.com/kanisterio/safecli"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/command"

	flagcommon "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/common"
	flagmanifest "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/manifest"
)

const (
	manifestFilterTypeSnapshot = "type:snapshot"
)

// ListArgs defines the arguments for the `kopia manifest list ...` command.
type ListArgs struct {
	cli.CommonArgs
}

// List creates a new `kopia manifest list ...` command.
func List(args ListArgs) (safecli.CommandBuilder, error) {
	return command.NewKopiaCommandBuilder(args.CommonArgs,
		command.Manifest, command.List,
		flagcommon.JSON,
		flagmanifest.Filter(manifestFilterTypeSnapshot),
	)
}
