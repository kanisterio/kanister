package repository

import (
	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/command"
	flagcommon "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/common"
	flagrepo "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/repository"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/safecli"
)

// ConnectServerArgs defines the arguments for the `kopia repository connect server` command.
type ConnectServerArgs struct {
	cli.CommonArgs
	cli.CacheArgs

	Hostname    string // hostname of the repository
	Username    string // username of the repository
	ServerURL   string // URL of the Kopia Repository API server
	Fingerprint string // fingerprint of the server's TLS certificate
	ReadOnly    bool   // connect to a repository in read-only mode

	Logger log.Logger
}

// ConnectServer creates a new `kopia repository connect server...` command.
func ConnectServer(args ConnectServerArgs) (safecli.CommandBuilder, error) {
	return command.NewKopiaCommandBuilder(args.CommonArgs,
		command.Repository, command.Connect, command.Server,
		flagcommon.NoCheckForUpdates,
		flagcommon.NoGRPC,
		flagcommon.ReadOnly(args.ReadOnly),
		flagcommon.Cache(args.CacheArgs),
		flagrepo.Hostname(args.Hostname),
		flagrepo.Username(args.Username),
		flagrepo.ServerURL(args.ServerURL),
		flagrepo.ServerCertFingerprint(args.Fingerprint),
	)
}
