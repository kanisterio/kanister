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
func RepositoryConnect(repositoryConnectArgs RepositoryConnectCommandArgs) ([]string, error) {
	args := commonArgs(repositoryConnectArgs.EncryptionKey, repositoryConnectArgs.ConfigFilePath, repositoryConnectArgs.LogDirectory, false)
	args = args.AppendLoggable(repositorySubCommand, connectSubCommand, noCheckForUpdatesFlag)

	args = kopiaCacheArgs(args, repositoryConnectArgs.CacheDirectory, repositoryConnectArgs.ContentCacheMB, repositoryConnectArgs.MetadataCacheMB)

	if repositoryConnectArgs.Hostname != "" {
		args = args.AppendLoggableKV(overrideHostnameFlag, repositoryConnectArgs.Hostname)
	}

	if repositoryConnectArgs.Username != "" {
		args = args.AppendLoggableKV(overrideUsernameFlag, repositoryConnectArgs.Username)
	}

	bsArgs, err := kopiaBlobStoreArgs(repositoryConnectArgs.Prof, repositoryConnectArgs.RepoPathPrefix)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to generate blob store args")
	}

	if !time.Time(repositoryConnectArgs.PointInTimeConnection).IsZero() {
		bsArgs = bsArgs.AppendLoggableKV(pointInTimeConnectionFlag, repositoryConnectArgs.PointInTimeConnection.String())
	}

	return stringSliceCommand(args.Combine(bsArgs)), nil
}
