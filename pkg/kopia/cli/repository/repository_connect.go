package repository

import (
	"github.com/go-openapi/strfmt"
	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/command"
	flagcommon "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/common"
	flagrepo "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/repository"
	flagstorage "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/storage"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/safecli"
)

// ConnectArgs defines the arguments for the `kopia repository connect` command.
type ConnectArgs struct {
	cli.CommonArgs
	cli.CacheArgs

	Hostname       string            // the hostname of the repository
	Username       string            // the username of the repository
	Location       map[string][]byte // the location of the repository
	RepoPathPrefix string            // the prefix of the repository path
	ReadOnly       bool              // connect to a repository in read-only mode
	PointInTime    strfmt.DateTime   // connect to a repository as it was at a specific point in time

	Logger log.Logger
}

// Connect creates a new `kopia repository connect ...` command.
func Connect(args ConnectArgs) (safecli.CommandBuilder, error) {
	return command.NewKopiaCommandBuilder(args.CommonArgs,
		command.Repository, command.Connect,
		flagcommon.NoCheckForUpdates,
		flagcommon.ReadOnly(args.ReadOnly),
		flagcommon.Cache(args.CacheArgs),
		flagrepo.Hostname(args.Hostname),
		flagrepo.Username(args.Username),
		flagstorage.Storage(args.Location, args.RepoPathPrefix, flagstorage.WithLogger(args.Logger)),
		flagrepo.PIT(args.PointInTime),
	)
}
