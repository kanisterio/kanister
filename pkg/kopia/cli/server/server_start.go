package server

import (
	"github.com/kanisterio/safecli"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/command"

	flagcommon "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/common"
	flagserver "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/server"
)

// StartArgs defines arguments for 'kopia server start ...' command.
// By default generated command will have bash wrapper.
type StartArgs struct {
	cli.CommonArgs

	TLSCertFile        string // the TLS certificate file
	TLSKeyFile         string // the TLS key file
	ServerAddress      string // the kopia server address
	ServerUsername     string // the username for the kopia server
	ServerPassword     string // the password for the kopia server
	AutoGenerateCert   bool   // auto generate TLS certificate
	Background         bool   // run the server in background
	DisableBashWrapper bool   // disable bash wrapper
}

// Create creates a new 'kopia server start ...' command.
func Create(args StartArgs) (safecli.CommandBuilder, error) {
	// create a new serverCommonArgs with the common args for kopia server start command
	// password and other args will be handled below.
	serverCommonArgs := cli.CommonArgs{
		ConfigFilePath: args.ConfigFilePath,
		LogDirectory:   args.LogDirectory,
	}

	start, err := command.NewKopiaCommandBuilder(serverCommonArgs,
		command.Server, command.Start,
		flagserver.TLSGenerateCert(args.AutoGenerateCert),
		flagserver.TLSCertFile(args.TLSCertFile),
		flagserver.TLSKeyFile(args.TLSKeyFile),
		flagserver.ServerAddress(args.ServerAddress),
		flagserver.ServerUsername(args.ServerUsername),
		flagserver.ServerPassword(args.ServerPassword),
		flagserver.ServerControlUsername(args.ServerUsername),
		flagserver.ServerControlPassword(args.ServerPassword),
		flagcommon.NoGRPC,
		flagserver.Background(args.Background),
	)
	if err != nil {
		return nil, err
	}

	if args.DisableBashWrapper {
		return start, nil
	}
	return NewBashBuilder(start), nil
}
