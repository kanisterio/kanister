package command

import (
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"

	"github.com/kanisterio/kanister/pkg/kopia/command/storage"
)

// RepositoryCommandArgs contains fields that are needed for
// creating or connecting to a Kopia repository
type RepositoryCommandArgs struct {
	*CommandArgs
	LocationSecret  *v1.Secret
	CredsSecret     *v1.Secret
	CacheDirectory  string
	Hostname        string
	ContentCacheMB  int
	MetadataCacheMB int
	Username        string
	RepoPathPrefix  string
	// PITFlag is only effective if set while repository connect
	PITFlag strfmt.DateTime
}

// RepositoryConnectCommand returns the kopia command for connecting to an existing repo
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
		RepoPathPrefix:     cmdArgs.RepoPathPrefix,
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to generate storage args")
	}

	if !time.Time(cmdArgs.PITFlag).IsZero() {
		bsArgs = bsArgs.AppendLoggableKV(pointInTimeConnectionFlag, cmdArgs.PITFlag.String())
	}

	return stringSliceCommand(args.Combine(bsArgs)), nil
}

// RepositoryCreateCommand returns the kopia command for creation of a repo
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
		RepoPathPrefix:     cmdArgs.RepoPathPrefix,
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to generate storage args")
	}

	return stringSliceCommand(args.Combine(bsArgs)), nil
}

// RepositoryServerCommandArgs contains fields required for connecting
// to Kopia Repository API server
type RepositoryServerCommandArgs struct {
	UserPassword    string
	ConfigFilePath  string
	LogDirectory    string
	CacheDirectory  string
	Hostname        string
	ServerURL       string
	Fingerprint     string
	Username        string
	ContentCacheMB  int
	MetadataCacheMB int
}

// RepositoryConnectServerCommand returns the kopia command for connecting to a remote
// repository on Kopia Repository API server
func RepositoryConnectServerCommand(cmdArgs RepositoryServerCommandArgs) []string {
	args := commonArgs(&CommandArgs{
		RepoPassword:   cmdArgs.UserPassword,
		ConfigFilePath: cmdArgs.ConfigFilePath,
		LogDirectory:   cmdArgs.LogDirectory,
	}, false)
	args = args.AppendLoggable(repositorySubCommand, connectSubCommand, serverSubCommand, noCheckForUpdatesFlag, noGrpcFlag)

	args = kopiaCacheArgs(args, cmdArgs.CacheDirectory, cmdArgs.ContentCacheMB, cmdArgs.MetadataCacheMB)

	if cmdArgs.Hostname != "" {
		args = args.AppendLoggableKV(overrideHostnameFlag, cmdArgs.Hostname)
	}

	if cmdArgs.Username != "" {
		args = args.AppendLoggableKV(overrideUsernameFlag, cmdArgs.Username)
	}
	args = args.AppendLoggableKV(urlFlag, cmdArgs.ServerURL)

	args = args.AppendRedactedKV(serverCertFingerprint, cmdArgs.Fingerprint)

	return stringSliceCommand(args)
}
