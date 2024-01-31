package server

import (
	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/command"
	flagserver "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/server"
	"github.com/kanisterio/kanister/pkg/safecli"
)

// StatusArgs defines the arguments for the 'kopia server status' command.
type StatusArgs struct {
	cli.CommonArgs

	ServerAddress  string // the kopia server address
	ServerUsername string // the username for the kopia server
	ServerPassword string // the password for the kopia server
	Fingerprint    string // server certificate fingerprint
}

// Status creates a new 'kopia server status' command.
func Status(args StatusArgs) (safecli.CommandBuilder, error) {
	// create a new serverCommonArgs with the common args for kopia server status command
	// password and other args will be handled below.
	serverCommonArgs := cli.CommonArgs{
		ConfigFilePath: args.ConfigFilePath,
		LogDirectory:   args.LogDirectory,
	}

	return command.NewKopiaCommandBuilder(serverCommonArgs,
		command.Server, command.Status,
		flagserver.ServerAddress(args.ServerAddress),
		flagserver.ServerUsername(args.ServerUsername),
		flagserver.ServerPassword(args.ServerPassword),
		flagserver.ServerCertFingerprint(args.Fingerprint),
	)
}
