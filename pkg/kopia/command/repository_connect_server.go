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

type RepositoryConnectServerCommandArgs struct {
	*CommandArgs
	cacheDirectory  string
	hostname        string
	serverURL       string
	fingerprint     string
	username        string
	userPassword    string
	contentCacheMB  int
	metadataCacheMB int
}

// RepositoryConnectServer returns the kopia command for connecting to a remote repository on Kopia API server
func RepositoryConnectServer(repositoryConnectServerArgs RepositoryConnectServerCommandArgs) []string {
	args := commonArgs(repositoryConnectServerArgs.userPassword, repositoryConnectServerArgs.configFilePath, repositoryConnectServerArgs.logDirectory, false)
	args = args.AppendLoggable(repositorySubCommand, connectSubCommand, serverSubCommand, noCheckForUpdatesFlag, noGrpcFlag)

	args = kopiaCacheArgs(args, repositoryConnectServerArgs.cacheDirectory, repositoryConnectServerArgs.contentCacheMB, repositoryConnectServerArgs.metadataCacheMB)

	if repositoryConnectServerArgs.hostname != "" {
		args = args.AppendLoggableKV(overrideHostnameFlag, repositoryConnectServerArgs.hostname)
	}

	if repositoryConnectServerArgs.username != "" {
		args = args.AppendLoggableKV(overrideUsernameFlag, repositoryConnectServerArgs.username)
	}
	args = args.AppendLoggableKV(urlFlag, repositoryConnectServerArgs.serverURL)

	args = args.AppendRedactedKV(serverCertFingerprint, repositoryConnectServerArgs.fingerprint)

	return stringSliceCommand(args)
}
