package gcs

import (
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/safecli"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/command"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/storage/model"
)

// New returns a builder for the GCS subcommand storage.
func New(s model.StorageFlag) (*safecli.Builder, error) {
	prefix := model.GenerateFullRepoPath(s.Location.Prefix(), s.RepoPathPrefix)
	return command.NewCommandBuilder(command.GCS,
		Bucket(s.Location.BucketName()),
		CredentialsFile(consts.GoogleCloudCredsFilePath),
		Prefix(prefix),
	)
}
