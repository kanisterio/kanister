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

type ServerRefreshCommandArgs struct {
	*CommandArgs
	serverAddress  string
	serverUsername string
	serverPassword string
	fingerprint    string
}

// ServerRefresh returns the kopia command for refreshing the Kopia API Server
// This helps allow new users to be able to connect to the Server instead of waiting for auto-refresh
func ServerRefresh(serverRefreshArgs ServerRefreshCommandArgs) []string {
	args := commonArgs(serverRefreshArgs.encryptionKey, serverRefreshArgs.configFilePath, serverRefreshArgs.logDirectory, false)
	args = args.AppendLoggable(serverSubCommand, refreshSubCommand)
	args = args.AppendRedactedKV(serverCertFingerprint, serverRefreshArgs.fingerprint)
	args = args.AppendLoggableKV(addressFlag, serverRefreshArgs.serverAddress)
	args = args.AppendLoggableKV(serverUsernameFlag, serverRefreshArgs.serverUsername)
	args = args.AppendRedactedKV(serverPasswordFlag, serverRefreshArgs.serverPassword)

	return stringSliceCommand(args)
}
