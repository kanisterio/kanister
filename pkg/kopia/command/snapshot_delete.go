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

type SnapshotDeleteCommandArgs struct {
	*CommandArgs
	snapID string
}

// SnapshotDelete returns the kopia command for deleting a snapshot with given snapshot ID
func SnapshotDelete(snapshotDeleteArgs SnapshotDeleteCommandArgs) []string {
	args := commonArgs(snapshotDeleteArgs.encryptionKey, snapshotDeleteArgs.configFilePath, snapshotDeleteArgs.logDirectory, false)
	args = args.AppendLoggable(snapshotSubCommand, deleteSubCommand, snapshotDeleteArgs.snapID, unsafeIgnoreSourceFlag)

	return stringSliceCommand(args)
}
