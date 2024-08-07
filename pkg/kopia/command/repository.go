// Copyright 2022 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package command

import (
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/kanisterio/errkit"

	"github.com/kanisterio/kanister/pkg/kopia/cli/args"
	"github.com/kanisterio/kanister/pkg/kopia/command/storage"
)

// RepositoryCommandArgs contains fields that are needed for
// creating or connecting to a Kopia repository
type RepositoryCommandArgs struct {
	*CommandArgs
	CacheArgs
	CacheDirectory  string
	Hostname        string
	ContentCacheMB  int
	MetadataCacheMB int
	Username        string
	RepoPathPrefix  string
	ReadOnly        bool
	// Only for CreateCommand
	RetentionMode   string
	RetentionPeriod time.Duration
	// PITFlag is only effective if set while repository connect
	PITFlag  strfmt.DateTime
	Location map[string][]byte
}

// RepositoryConnectCommand returns the kopia command for connecting to an existing repo
func RepositoryConnectCommand(cmdArgs RepositoryCommandArgs) ([]string, error) {
	args := commonArgs(cmdArgs.CommandArgs)
	args = args.AppendLoggable(repositorySubCommand, connectSubCommand, noCheckForUpdatesFlag)

	if cmdArgs.ReadOnly {
		args = args.AppendLoggable(readOnlyFlag)
	}

	args = cmdArgs.kopiaCacheArgs(args, cmdArgs.CacheDirectory)

	if cmdArgs.Hostname != "" {
		args = args.AppendLoggableKV(overrideHostnameFlag, cmdArgs.Hostname)
	}

	if cmdArgs.Username != "" {
		args = args.AppendLoggableKV(overrideUsernameFlag, cmdArgs.Username)
	}

	bsArgs, err := storage.KopiaStorageArgs(&storage.StorageCommandParams{
		Location:       cmdArgs.Location,
		RepoPathPrefix: cmdArgs.RepoPathPrefix,
	})
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to generate storage args")
	}

	if !time.Time(cmdArgs.PITFlag).IsZero() {
		bsArgs = bsArgs.AppendLoggableKV(pointInTimeConnectionFlag, cmdArgs.PITFlag.String())
	}

	return stringSliceCommand(args.Combine(bsArgs)), nil
}

// RepositoryCreateCommand returns the kopia command for creation of a repo
func RepositoryCreateCommand(cmdArgs RepositoryCommandArgs) ([]string, error) {
	command := commonArgs(cmdArgs.CommandArgs)
	command = command.AppendLoggable(repositorySubCommand, createSubCommand, noCheckForUpdatesFlag)

	command = cmdArgs.kopiaCacheArgs(command, cmdArgs.CacheDirectory)

	if cmdArgs.Hostname != "" {
		command = command.AppendLoggableKV(overrideHostnameFlag, cmdArgs.Hostname)
	}

	if cmdArgs.Username != "" {
		command = command.AppendLoggableKV(overrideUsernameFlag, cmdArgs.Username)
	}

	// During creation, both should be set. Technically RetentionPeriod should be >= 24 * time.Hour
	if cmdArgs.RetentionMode != "" && cmdArgs.RetentionPeriod > 0 {
		command = command.AppendLoggableKV(retentionModeFlag, cmdArgs.RetentionMode)
		command = command.AppendLoggableKV(retentionPeriodFlag, cmdArgs.RetentionPeriod.String())
	}

	command = args.RepositoryCreate.AppendToCmd(command)

	bsArgs, err := storage.KopiaStorageArgs(&storage.StorageCommandParams{
		Location:       cmdArgs.Location,
		RepoPathPrefix: cmdArgs.RepoPathPrefix,
	})
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to generate storage args")
	}

	return stringSliceCommand(command.Combine(bsArgs)), nil
}

// RepositoryServerCommandArgs contains fields required for connecting
// to Kopia Repository API server
type RepositoryServerCommandArgs struct {
	UserPassword   string
	ConfigFilePath string
	LogDirectory   string
	CacheDirectory string
	Hostname       string
	ServerURL      string
	Fingerprint    string
	Username       string
	ReadOnly       bool
	CacheArgs
}

// RepositoryConnectServerCommand returns the kopia command for connecting to a remote
// repository on Kopia Repository API server
func RepositoryConnectServerCommand(cmdArgs RepositoryServerCommandArgs) []string {
	command := commonArgs(&CommandArgs{
		RepoPassword:   cmdArgs.UserPassword,
		ConfigFilePath: cmdArgs.ConfigFilePath,
		LogDirectory:   cmdArgs.LogDirectory,
	})
	command = command.AppendLoggable(repositorySubCommand, connectSubCommand, serverSubCommand, noCheckForUpdatesFlag)

	if cmdArgs.ReadOnly {
		command = command.AppendLoggable(readOnlyFlag)
	}

	command = cmdArgs.kopiaCacheArgs(command, cmdArgs.CacheDirectory)

	if cmdArgs.Hostname != "" {
		command = command.AppendLoggableKV(overrideHostnameFlag, cmdArgs.Hostname)
	}

	if cmdArgs.Username != "" {
		command = command.AppendLoggableKV(overrideUsernameFlag, cmdArgs.Username)
	}
	command = command.AppendLoggableKV(urlFlag, cmdArgs.ServerURL)

	command = args.RepositoryConnectServer.AppendToCmd(command)

	command = command.AppendRedactedKV(serverCertFingerprint, cmdArgs.Fingerprint)

	return stringSliceCommand(command)
}

type RepositoryStatusCommandArgs struct {
	*CommandArgs
	GetJSONOutput bool
}

// RepositoryStatusCommand returns the kopia command for checking status of the Kopia repository
func RepositoryStatusCommand(cmdArgs RepositoryStatusCommandArgs) []string {
	// Default to info log level unless specified otherwise.
	if cmdArgs.LogLevel == "" {
		// Make a copy of the common command args, set the log level to info.
		common := *cmdArgs.CommandArgs
		common.LogLevel = LogLevelInfo
		cmdArgs.CommandArgs = &common
	}

	args := commonArgs(cmdArgs.CommandArgs)
	args = args.AppendLoggable(repositorySubCommand, statusSubCommand)
	if cmdArgs.GetJSONOutput {
		args = args.AppendLoggable(jsonFlag)
	}

	return stringSliceCommand(args)
}

type RepositorySetParametersCommandArgs struct {
	*CommandArgs
	RetentionMode   string
	RetentionPeriod time.Duration
}

// RepositorySetParametersCommand to cover https://kopia.io/docs/reference/command-line/common/repository-set-parameters/
func RepositorySetParametersCommand(cmdArgs RepositorySetParametersCommandArgs) []string {
	args := commonArgs(cmdArgs.CommandArgs)
	args = args.AppendLoggable(repositorySubCommand, setParametersSubCommand)
	// RetentionPeriod can be 0 when wanting to disable blob retention or when changing the mode only
	if cmdArgs.RetentionMode != "" {
		args = args.AppendLoggableKV(retentionModeFlag, cmdArgs.RetentionMode)
		args = args.AppendLoggableKV(retentionPeriodFlag, cmdArgs.RetentionPeriod.String())
	}
	return stringSliceCommand(args)
}
