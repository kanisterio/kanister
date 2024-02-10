package server

import (
	"github.com/kanisterio/safecli"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/command"

	flagserver "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/server"
)

// UserSetArgs defines the arguments for the 'kopia server set user' subcommand.
type UserSetArgs struct {
	cli.CommonArgs

	Username     string // the username for the kopia server
	UserPassword string // the password for the kopia user
}

func UserSet(args UserSetArgs) (safecli.CommandBuilder, error) {
	return command.NewKopiaCommandBuilder(args.CommonArgs,
		command.Server, command.User, command.Set,
		flagserver.Username(args.Username),
		flagserver.UserPassword(args.UserPassword),
	)
}
