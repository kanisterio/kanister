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

// RepositoryConnect returns the kopia command for connecting to an existing blob-store repo
func RepositoryConnect(
	encryptionKey,
	configFilePath,
	logDirectory string,
	prof kopia.Profile,
	artifactPrefix,
	hostname,
	username,
	cacheDirectory string,
	contentCacheMB,
	metadataCacheMB int,
	pointInTimeConnection strfmt.DateTime,
) ([]string, error) {
	args := commonArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(repositorySubCommand, connectSubCommand, noCheckForUpdatesFlag)

	args = kopiaCacheArgs(args, cacheDirectory, contentCacheMB, metadataCacheMB)

	if hostname != "" {
		args = args.AppendLoggableKV(overrideHostnameFlag, hostname)
	}

	if username != "" {
		args = args.AppendLoggableKV(overrideUsernameFlag, username)
	}

	bsArgs, err := kopiaBlobStoreArgs(prof, artifactPrefix)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to generate blob store args")
	}

	if !time.Time(pointInTimeConnection).IsZero() {
		bsArgs = bsArgs.AppendLoggableKV(pointInTimeConnectionFlag, pointInTimeConnection.String())
	}

	return stringSliceCommand(args.Combine(bsArgs)), nil
}
