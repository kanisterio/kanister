package azure

import (
	"github.com/kanisterio/kanister/pkg/safecli"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/command"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/storage/model"
)

// New returns a builder for the Azure subcommand storage.
func New(f model.StorageFlag) (*safecli.Builder, error) {
	prefix := model.GenerateFullRepoPath(f.Location.Prefix(), f.RepoPathPrefix)
	return command.NewCommandBuilder(command.Azure,
		AzureCountainer(f.Location.BucketName()),
		Prefix(prefix),
	)
}
