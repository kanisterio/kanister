package command

import (
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"

	"github.com/kanisterio/kanister/pkg/kopia/command/storage"
)

const (
	repositorySubCommand      = "repository"
	connectSubCommand         = "connect"
	noCheckForUpdatesFlag     = "--no-check-for-updates"
	overrideHostnameFlag      = "--override-hostname"
	overrideUsernameFlag      = "--override-username"
	pointInTimeConnectionFlag = "--point-in-time"
)

type RepositoryCommandArgs struct {
	*CommandArgs
	LocationSecret  *v1.Secret
	CredsSecret     *v1.Secret
	CacheDirectory  string
	Hostname        string
	ContentCacheMB  int
	MetadataCacheMB int
	Username        string
	ArtifactPrefix  string
	PITFlag         strfmt.DateTime
}

// RepositoryConnectCommand returns the kopia command for connecting to an existing blob-store repo
func RepositoryConnectCommand(cmdArgs RepositoryCommandArgs) ([]string, error) {
	args := commonArgs(cmdArgs.CommandArgs, false)
	args = args.AppendLoggable(repositorySubCommand, connectSubCommand, noCheckForUpdatesFlag)

	args = kopiaCacheArgs(args, cmdArgs.CacheDirectory, cmdArgs.ContentCacheMB, cmdArgs.MetadataCacheMB)

	if cmdArgs.Hostname != "" {
		args = args.AppendLoggableKV(overrideHostnameFlag, cmdArgs.Hostname)
	}

	if cmdArgs.Username != "" {
		args = args.AppendLoggableKV(overrideUsernameFlag, cmdArgs.Username)
	}

	bsArgs, err := storage.KopiaBlobStoreArgs(&storage.StorageCommandParams{
		LocationSecret:     cmdArgs.LocationSecret,
		LocationCredSecret: cmdArgs.CredsSecret,
		ArtifactPrefix:     cmdArgs.ArtifactPrefix,
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to generate blob store args")
	}

	if !time.Time(cmdArgs.PITFlag).IsZero() {
		bsArgs = bsArgs.AppendLoggableKV(pointInTimeConnectionFlag, cmdArgs.PITFlag.String())
	}

	return stringSliceCommand(args.Combine(bsArgs)), nil
}

// RepositoryCreateCommand returns the kopia command for creation of a blob-store repo
func RepositoryCreateCommand(cmdArgs RepositoryCommandArgs) ([]string, error) {
	args := commonArgs(cmdArgs.CommandArgs, false)
	args = args.AppendLoggable(repositorySubCommand, createSubCommand, noCheckForUpdatesFlag)

	args = kopiaCacheArgs(args, cmdArgs.CacheDirectory, cmdArgs.ContentCacheMB, cmdArgs.MetadataCacheMB)

	if cmdArgs.Hostname != "" {
		args = args.AppendLoggableKV(overrideHostnameFlag, cmdArgs.Hostname)
	}

	if cmdArgs.Username != "" {
		args = args.AppendLoggableKV(overrideUsernameFlag, cmdArgs.Username)
	}

	bsArgs, err := storage.KopiaBlobStoreArgs(&storage.StorageCommandParams{
		LocationSecret:     cmdArgs.LocationSecret,
		LocationCredSecret: cmdArgs.CredsSecret,
		ArtifactPrefix:     cmdArgs.ArtifactPrefix,
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to generate blob store args")
	}

	return stringSliceCommand(args.Combine(bsArgs)), nil
}
