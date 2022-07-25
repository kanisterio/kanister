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
	"github.com/pkg/errors"

	"github.com/kanisterio/kanister/pkg/kopia"
)

// RepositoryCreate returns the kopia command for creation of a blob-store repo
// TODO: Consolidate all the repository options into a struct and pass
func RepositoryCreate(
	configFilePath,
	logDirectory string,
	prof kopia.Profile,
	artifactPrefix,
	encryptionKey,
	hostname,
	username,
	cacheDirectory string,
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

	bsArgs, err := kopiaBlobStoreArgs(prof, artifactPrefix)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to generate blob store args")
	}

	return stringSliceCommand(args.Combine(bsArgs)), nil
}
