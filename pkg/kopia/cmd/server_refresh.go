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

// ServerRefresh returns the kopia command for refreshing the Kopia API Server
// This helps allow new users to be able to connect to the Server instead of waiting for auto-refresh
func ServerRefresh(
	encryptionKey,
	configFilePath,
	logDirectory,
	serverAddress,
	serverUsername,
	serverPassword,
	fingerprint string,
) []string {
	args := commonArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(serverSubCommand, refreshSubCommand)
	args = args.AppendRedactedKV(serverCertFingerprint, fingerprint)
	args = args.AppendLoggableKV(addressFlag, serverAddress)
	args = args.AppendLoggableKV(serverUsernameFlag, serverUsername)
	args = args.AppendRedactedKV(serverPasswordFlag, serverPassword)

	return stringSliceCommand(args)
}
