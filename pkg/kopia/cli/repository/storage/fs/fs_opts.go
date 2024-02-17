package fs

import (
	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/safecli/command"
)

var (
	subcmdFilesystem = command.NewArgument("filesystem")
)

// optRepoPath creates a new path option with a given path.
// If the path is empty, it returns an error.
func optRepoPath(path string) command.Applier {
	if path == "" {
		return command.NewErrorArgument(cli.ErrInvalidRepoPath)
	}
	return command.NewOptionWithArgument("--path", path)
}
