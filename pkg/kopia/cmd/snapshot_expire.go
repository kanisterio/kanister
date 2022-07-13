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

// SnapshotExpire returns the kopia command for removing snapshots with given root ID
func SnapshotExpire(encryptionKey, rootID, configFilePath, logDirectory string, mustDelete bool) []string {
	return stringSliceCommand(snapshotExpire(encryptionKey, rootID, configFilePath, logDirectory, mustDelete))
}

func snapshotExpire(encryptionKey, rootID, configFilePath, logDirectory string, mustDelete bool) logsafe.Cmd {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(snapshotSubCommand, expireSubCommand, rootID)
	if mustDelete {
		args = args.AppendLoggable(deleteFlag)
	}

	return args
}
