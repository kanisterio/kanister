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

// SnapListAll returns the kopia command for listing all snapshots in the repository with their sizes
func SnapListAll(encryptionKey, configFilePath, logDirectory string) []string {
	return stringSliceCommand(snapListAll(encryptionKey, configFilePath, logDirectory))
}

func snapListAll(encryptionKey, configFilePath, logDirectory string) logsafe.Cmd {
	args := commonArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(
		snapshotSubCommand,
		listSubCommand,
		allFlag,
		deltaFlag,
		showIdenticalFlag,
		jsonFlag,
	)

	return args
}
