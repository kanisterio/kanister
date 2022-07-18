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

package cmd

import (
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/pkg/errors"

	"github.com/kanisterio/kanister/pkg/kopia"
	"github.com/kanisterio/kanister/pkg/logsafe"
)

// RepositoryConnect returns the kopia command for connecting to an existing blob-store repo
func RepositoryConnect(
	prof kopia.Profile,
	artifactPrefix,
	encryptionKey,
	hostname,
	username,
	cacheDirectory,
	configFilePath,
	logDirectory string,
	contentCacheMB,
	metadataCacheMB int,
	pointInTimeConnection strfmt.DateTime,
) ([]string, error) {
	cmd, err := repositoryConnect(
		prof,
		artifactPrefix,
		encryptionKey,
		hostname,
		username,
		cacheDirectory,
		configFilePath,
		logDirectory,
		contentCacheMB,
		metadataCacheMB,
		pointInTimeConnection,
	)
	if err != nil {
		return nil, err
	}

	return stringSliceCommand(cmd), nil
}

func repositoryConnect(
	prof kopia.Profile,
	artifactPrefix,
	encryptionKey,
	hostname,
	username,
	cacheDirectory,
	configFilePath,
	logDirectory string,
	contentCacheMB,
	metadataCacheMB int,
	pointInTimeConnection strfmt.DateTime,
) (logsafe.Cmd, error) {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
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

	return args.Combine(bsArgs), nil
}
