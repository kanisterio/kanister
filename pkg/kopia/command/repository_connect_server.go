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
func RepositoryConnectServer(repositoryConnectServerArgs RepositoryConnectServerCommandArgs) []string {
	args := commonArgs(repositoryConnectServerArgs.UserPassword, repositoryConnectServerArgs.ConfigFilePath, repositoryConnectServerArgs.LogDirectory, false)
	args = args.AppendLoggable(repositorySubCommand, connectSubCommand, serverSubCommand, noCheckForUpdatesFlag, noGrpcFlag)

	args = kopiaCacheArgs(args, repositoryConnectServerArgs.CacheDirectory, repositoryConnectServerArgs.ContentCacheMB, repositoryConnectServerArgs.MetadataCacheMB)

	if repositoryConnectServerArgs.Hostname != "" {
		args = args.AppendLoggableKV(overrideHostnameFlag, repositoryConnectServerArgs.Hostname)
	}

	if repositoryConnectServerArgs.Username != "" {
		args = args.AppendLoggableKV(overrideUsernameFlag, repositoryConnectServerArgs.Username)
	}
	args = args.AppendLoggableKV(urlFlag, repositoryConnectServerArgs.ServerURL)

	args = args.AppendRedactedKV(serverCertFingerprint, repositoryConnectServerArgs.Fingerprint)

	return stringSliceCommand(args)
}
