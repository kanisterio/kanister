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

// ServerListUser returns the kopia command to list users from the Kopia API Server
func ServerListUser(
	encryptionKey,
	configFilePath,
	logDirectory string,
) []string {
	return stringSliceCommand(serverListUser(
		encryptionKey,
		configFilePath,
		logDirectory,
	))
}

func serverListUser(
	encryptionKey,
	configFilePath,
	logDirectory string,
) logsafe.Cmd {
	args := commonArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(serverSubCommand, userSubCommand, listSubCommand, jsonFlag)

	return args
}
