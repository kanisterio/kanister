package snapshot

import (
	"github.com/kanisterio/safecli"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/command"

	flagcommon "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/common"
	flagsnapshot "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/snapshot"
)

// ListArgs defines the arguments for the `kopia snapshot list ...` command.
type ListArgs struct {
	cli.CommonArgs
	FilterByTags []string // filter by tags if set
}

// List creates a new `kopia snapshot list ...` command.
func List(args ListArgs) (safecli.CommandBuilder, error) {
	return command.NewKopiaCommandBuilder(args.CommonArgs,
		command.Snapshot, command.List,
		flagcommon.All,
		flagcommon.Delta,
		flagcommon.ShowIdentical,
		flagcommon.JSON,
		flagsnapshot.Tags(args.FilterByTags),
	)
}
