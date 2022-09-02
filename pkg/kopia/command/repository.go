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
	"github.com/pkg/errors"

	"github.com/kanisterio/kanister/pkg/kopia"
)

type RepositoryCreateCommandArgs struct {
	*CommandArgs
	Prof            kopia.Profile
	RepoPathPrefix  string
	Hostname        string
	Username        string
	CacheDirectory  string
	ContentCacheMB  int
	MetadataCacheMB int
}

// RepositoryCreate returns the kopia command for creation of a blob-store repo
func RepositoryCreate(repositoryCreateArgs RepositoryCreateCommandArgs) ([]string, error) {
	args := commonArgs(repositoryCreateArgs.EncryptionKey, repositoryCreateArgs.ConfigFilePath, repositoryCreateArgs.LogDirectory, false)
	args = args.AppendLoggable(repositorySubCommand, createSubCommand, noCheckForUpdatesFlag)

	args = kopiaCacheArgs(args, repositoryCreateArgs.CacheDirectory, repositoryCreateArgs.ContentCacheMB, repositoryCreateArgs.MetadataCacheMB)

	if repositoryCreateArgs.Hostname != "" {
		args = args.AppendLoggableKV(overrideHostnameFlag, repositoryCreateArgs.Hostname)
	}

	if repositoryCreateArgs.Username != "" {
		args = args.AppendLoggableKV(overrideUsernameFlag, repositoryCreateArgs.Username)
	}

	bsArgs, err := kopiaBlobStoreArgs(repositoryCreateArgs.Prof, repositoryCreateArgs.RepoPathPrefix)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to generate blob store args")
	}

	return stringSliceCommand(args.Combine(bsArgs)), nil
}

type RepositoryConnectCommandArgs struct {
	*CommandArgs
	Prof                  kopia.Profile
	RepoPathPrefix        string
	Hostname              string
	Username              string
	CacheDirectory        string
	ContentCacheMB        int
	MetadataCacheMB       int
	PointInTimeConnection strfmt.DateTime
}

// RepositoryConnect returns the kopia command for connecting to an existing blob-store repo
func RepositoryConnect(cmdArgs RepositoryConnectCommandArgs) ([]string, error) {
	args := commonArgs(cmdArgs.EncryptionKey, cmdArgs.ConfigFilePath, cmdArgs.LogDirectory, false)
	args = args.AppendLoggable(repositorySubCommand, connectSubCommand, noCheckForUpdatesFlag)

	args = kopiaCacheArgs(args, cmdArgs.CacheDirectory, cmdArgs.ContentCacheMB, cmdArgs.MetadataCacheMB)

	if cmdArgs.Hostname != "" {
		args = args.AppendLoggableKV(overrideHostnameFlag, cmdArgs.Hostname)
	}

	if cmdArgs.Username != "" {
		args = args.AppendLoggableKV(overrideUsernameFlag, cmdArgs.Username)
	}

	bsArgs, err := kopiaBlobStoreArgs(cmdArgs.Prof, cmdArgs.RepoPathPrefix)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to generate blob store args")
	}

	if !time.Time(cmdArgs.PointInTimeConnection).IsZero() {
		bsArgs = bsArgs.AppendLoggableKV(pointInTimeConnectionFlag, cmdArgs.PointInTimeConnection.String())
	}

	return stringSliceCommand(args.Combine(bsArgs)), nil
}

type RepositoryConnectServerCommandArgs struct {
	*CommandArgs
	CacheDirectory  string
	Hostname        string
	ServerURL       string
	Fingerprint     string
	Username        string
	UserPassword    string
	ContentCacheMB  int
	MetadataCacheMB int
}

// RepositoryConnectServer returns the kopia command for connecting to a remote repository on Kopia API server
func RepositoryConnectServer(cmdArgs RepositoryConnectServerCommandArgs) []string {
	args := commonArgs(cmdArgs.UserPassword, cmdArgs.ConfigFilePath, cmdArgs.LogDirectory, false)
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
