package fs

import (
	"github.com/kanisterio/kanister/pkg/safecli"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/command"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/storage/model"
)

const (
	DefaultFSMountPath = "/mnt/data"
)

// New returns a builder for the filesystem subcommand storage.
func New(f model.StorageFlag) (*safecli.Builder, error) {
	path := generateFileSystemMountPath(f.Location.Prefix(), f.RepoPathPrefix)
	return command.NewCommandBuilder(command.FileSystem,
		Path(path),
	)
}

func generateFileSystemMountPath(locPrefix, repoPathPrefix string) string {
	return DefaultFSMountPath + "/" + model.GenerateFullRepoPath(locPrefix, repoPathPrefix)
}
