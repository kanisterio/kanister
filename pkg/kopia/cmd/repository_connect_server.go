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

import "github.com/kanisterio/kanister/pkg/logsafe"

// RepositoryConnectServer returns the kopia command for connecting to a remote repository on Kopia API server
func RepositoryConnectServer(
	cacheDirectory,
	configFilePath,
	hostname,
	logDirectory,
	serverURL,
	fingerprint,
	username,
	userPassword string,
	contentCacheMB,
	metadataCacheMB int,
) []string {
	return stringSliceCommand(repositoryConnectServer(
		cacheDirectory,
		configFilePath,
		hostname,
		logDirectory,
		serverURL,
		fingerprint,
		username,
		userPassword,
		contentCacheMB,
		metadataCacheMB,
	))
}

func repositoryConnectServer(
	cacheDirectory,
	configFilePath,
	hostname,
	logDirectory,
	serverURL,
	fingerprint,
	username,
	userPassword string,
	contentCacheMB,
	metadataCacheMB int,
) logsafe.Cmd {
	args := kopiaArgs(userPassword, configFilePath, logDirectory, false)
	args = args.AppendLoggable(repositorySubCommand, connectSubCommand, serverSubCommand, noCheckForUpdatesFlag, noGrpcFlag)

	args = kopiaCacheArgs(args, cacheDirectory, contentCacheMB, metadataCacheMB)

	if hostname != "" {
		args = args.AppendLoggableKV(overrideHostnameFlag, hostname)
	}

	if username != "" {
		args = args.AppendLoggableKV(overrideUsernameFlag, username)
	}
	args = args.AppendLoggableKV(urlFlag, serverURL)

	args = args.AppendRedactedKV(serverCertFingerprint, fingerprint)

	return args
}
