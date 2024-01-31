package repository

import (
	"time"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/command"
	flagrepo "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/repository"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/safecli"
)

// SetParametersArgs defines the arguments for the `kopia repository set-parameters ...` command.
type SetParametersArgs struct {
	cli.CommonArgs

	RetentionMode   string        // retention mode for supported storage backends
	RetentionPeriod time.Duration // retention period for supported storage backends

	Logger log.Logger
}

// SetParameters creates a new `kopia repository set-parameters ...` command.
func SetParameters(args SetParametersArgs) (safecli.CommandBuilder, error) {
	return command.NewKopiaCommandBuilder(args.CommonArgs,
		command.Repository, command.SetParameters,
		flagrepo.BlobRetention(args.RetentionMode, args.RetentionPeriod),
	)
}
