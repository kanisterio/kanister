package server

import (
	"github.com/kanisterio/safecli"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/command"

	flagcommon "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/common"
)

// UserListArgs defines arguments for 'kopia server user list' command.
type UserListArgs struct {
	cli.CommonArgs
}

// UserList creates a new 'kopia server user list' command.
func UserList(args UserListArgs) (safecli.CommandBuilder, error) {
	return command.NewKopiaCommandBuilder(args.CommonArgs,
		command.Server, command.User, command.List,
		flagcommon.JSON,
	)
}
