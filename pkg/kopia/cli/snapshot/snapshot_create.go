package snapshot

import (
	"time"

	"github.com/kanisterio/safecli"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/command"

	flagcommon "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/common"
	flagsnapshot "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/snapshot"
)

// CreateArgs defines the arguments for the `kopia snapshot create ...` command.
type CreateArgs struct {
	cli.CommonArgs
	PathToBackup           string        // the path to backup
	Parallelism            int           // the number of parallel uploads
	ProgressUpdateInterval time.Duration // the progress update interval
	Tags                   []string      // the tags to apply to the snapshot
}

// Create creates a new `kopia snapshot create ...` command.
func Create(args CreateArgs) (safecli.CommandBuilder, error) {
	cmnArgs := args.CommonArgs
	if cmnArgs.LogLevel == "" {
		cmnArgs.LogLevel = "info"
	}
	return command.NewKopiaCommandBuilder(cmnArgs,
		command.Snapshot, command.Create,
		flagcommon.JSON,
		flagsnapshot.PathToBackup(args.PathToBackup),
		flagsnapshot.Parallel(args.Parallelism),
		flagsnapshot.ProgressUpdateInterval(args.ProgressUpdateInterval),
		flagsnapshot.Tags(args.Tags),
	)
}
