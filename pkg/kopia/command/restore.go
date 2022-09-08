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

import "github.com/kanisterio/kanister/pkg/logsafe"

type RestoreCommandArgs struct {
	*CommandArgs
	RootID     string
	TargetPath string
}

// Restore returns the kopia command for restoring root of a snapshot with given root ID
func Restore(cmdArgs RestoreCommandArgs) []string {
	return stringSliceCommand(restoreCommand(cmdArgs))
}

func restoreCommand(cmdArgs RestoreCommandArgs) logsafe.Cmd {
	args := commonArgs(cmdArgs.EncryptionKey, cmdArgs.ConfigFilePath, cmdArgs.LogDirectory, false)
	args = args.AppendLoggable(restoreSubCommand, cmdArgs.RootID, cmdArgs.TargetPath)
	return args
}
