package fs

import (
	"github.com/kanisterio/safecli/command"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal"
	"github.com/kanisterio/kanister/pkg/log"
)

const (
	defaultFSMountPath = "/mnt/data"
)

// New creates a new subcommand for the filesystem storage.
func New(location internal.Location, repoPathPrefix string, _ log.Logger) command.Applier {
	path, err := generateFileSystemMountPath(location.Prefix(), repoPathPrefix)
	if err != nil {
		return command.NewErrorArgument(err)
	}
	return command.NewArguments(subcmdFilesystem, optRepoPath(path))
}

// generateFileSystemMountPath generates the mount path for the filesystem storage.
func generateFileSystemMountPath(locPrefix, repoPrefix string) (string, error) {
	fullRepoPath := internal.GenerateFullRepoPath(locPrefix, repoPrefix)
	if fullRepoPath == "" {
		return "", cli.ErrInvalidRepoPath
	}
	return defaultFSMountPath + "/" + fullRepoPath, nil
}
