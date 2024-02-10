package server

import (
	"github.com/kanisterio/safecli"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/command"

	flagserver "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/server"
)

// UserAddArgs defines the arguments for the 'kopia server user add' subcommand.
type UserAddArgs struct {
	cli.CommonArgs
	Username     string // the username for the kopia server
	UserPassword string // the password for the kopia user
}

// UserAdd creates a new 'kopia server user add' command.
func UserAdd(args UserAddArgs) (safecli.CommandBuilder, error) {
	return command.NewKopiaCommandBuilder(args.CommonArgs,
		command.Server, command.User, command.Add,
		flagserver.Username(args.Username),
		flagserver.UserPassword(args.UserPassword),
	)
}
