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

// RepositoryConnectCommand returns the kopia command for connecting to an existing blob-store repo
func RepositoryConnectCommand(
	locationSecret,
	locationCredSecret *v1.Secret,
	artifactPrefix,
	repoPassword,
	hostname,
	username,
	cacheDirectory,
	configFilePath,
	logDirectory string,
	contentCacheMB,
	metadataCacheMB int,
	pointInTimeConnection strfmt.DateTime,
) ([]string, error) {
	args := commonArgs(repoPassword, configFilePath, logDirectory, false)
	args = args.AppendLoggable(repositorySubCommand, connectSubCommand, noCheckForUpdatesFlag)

	args = kopiaCacheArgs(args, cacheDirectory, contentCacheMB, metadataCacheMB)

	if hostname != "" {
		args = args.AppendLoggableKV(overrideHostnameFlag, hostname)
	}

	if username != "" {
		args = args.AppendLoggableKV(overrideUsernameFlag, username)
	}

	bsArgs, err := storage.KopiaBlobStoreArgs(&storage.StorageCommandParams{
		LocationSecret:     locationSecret,
		LocationCredSecret: locationCredSecret,
		ArtifactPrefix:     artifactPrefix,
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to generate blob store args")
	}

	if !time.Time(pointInTimeConnection).IsZero() {
		bsArgs = bsArgs.AppendLoggableKV(pointInTimeConnectionFlag, pointInTimeConnection.String())
	}

	return stringSliceCommand(args.Combine(bsArgs)), nil
}

// RepositoryCreateCommand returns the kopia command for creation of a blob-store repo
func RepositoryCreateCommand(
	locationSecret,
	locationCredSecret *v1.Secret,
	artifactPrefix,
	encryptionKey,
	hostname,
	username,
	cacheDirectory,
	configFilePath,
	logDirectory string,
	contentCacheMB,
	metadataCacheMB int,
) ([]string, error) {
	args := commonArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(repositorySubCommand, createSubCommand, noCheckForUpdatesFlag)

	args = kopiaCacheArgs(args, cacheDirectory, contentCacheMB, metadataCacheMB)

	if hostname != "" {
		args = args.AppendLoggableKV(overrideHostnameFlag, hostname)
	}

	if username != "" {
		args = args.AppendLoggableKV(overrideUsernameFlag, username)
	}

	bsArgs, err := storage.KopiaBlobStoreArgs(&storage.StorageCommandParams{
		LocationSecret:     locationSecret,
		LocationCredSecret: locationCredSecret,
		ArtifactPrefix:     artifactPrefix,
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to generate blob store args")
	}

	return stringSliceCommand(args.Combine(bsArgs)), nil
}
