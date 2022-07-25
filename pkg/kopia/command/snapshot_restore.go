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

// SnapshotRestore returns kopia command restoring snapshots with given snap ID
func SnapshotRestore(encryptionKey, configFilePath, logDirectory, snapID, targetPath string, sparseRestore bool) []string {
	args := commonArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(snapshotSubCommand, restoreSubCommand, snapID, targetPath)
	if sparseRestore {
		args = args.AppendLoggable(sparseFlag)
	}

	return stringSliceCommand(args)
}
