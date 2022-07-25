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

import (
	"github.com/kanisterio/kanister/pkg/kopia"
)

type SnapListAllWithSnapIDsCommandArgs struct {
	*CommandArgs
}

// SnapListAllWithSnapIDs returns the kopia command for listing all snapshots in the repository with snapshotIDs
func SnapListAllWithSnapIDs(snapListAllWithSnapIDsArgs SnapListAllWithSnapIDsCommandArgs) []string {
	args := commonArgs(snapListAllWithSnapIDsArgs.EncryptionKey, snapListAllWithSnapIDsArgs.ConfigFilePath, snapListAllWithSnapIDsArgs.LogDirectory, false)
	args = args.AppendLoggable(manifestSubCommand, listSubCommand, jsonFlag)
	args = args.AppendLoggableKV(filterFlag, kopia.ManifestTypeSnapshotFilter)

	return stringSliceCommand(args)
}
