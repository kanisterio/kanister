package server

import (
	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/command"
	"github.com/kanisterio/kanister/pkg/safecli"

	flagserver "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/server"
)

// RefreshArgs defines the arguments for the 'kopia server refresh' command.
type RefreshArgs struct {
	cli.CommonArgs

	ServerAddress  string // the kopia server address
	ServerUsername string // the username for the kopia server
	ServerPassword string // the password for the kopia server
	Fingerprint    string // server certificate fingerprint
}

// Refresh creates a new 'kopia server refresh' command.
func Refresh(args RefreshArgs) (safecli.CommandBuilder, error) {
	return command.NewKopiaCommandBuilder(args.CommonArgs,
		command.Server, command.Refresh,
		flagserver.ServerAddress(args.ServerAddress),
		flagserver.ServerUsername(args.ServerUsername),
		flagserver.ServerPassword(args.ServerPassword),
		flagserver.ServerCertFingerprint(args.Fingerprint),
	)
}
