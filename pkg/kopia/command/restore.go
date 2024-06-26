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

import "strconv"

type RestoreCommandArgs struct {
	*CommandArgs
	RootID                 string
	TargetPath             string
	IgnorePermissionErrors bool
	Parallelism            int
}

// Restore returns the kopia command for restoring root of a snapshot with given root ID
func Restore(cmdArgs RestoreCommandArgs) []string {
	args := commonArgs(cmdArgs.CommandArgs)
	args = args.AppendLoggable(restoreSubCommand, cmdArgs.RootID, cmdArgs.TargetPath)
	if cmdArgs.IgnorePermissionErrors {
		args = args.AppendLoggable(ignorePermissionsError)
	} else {
		args = args.AppendLoggable(noIgnorePermissionsError)
	}

	if cmdArgs.Parallelism > 0 {
		parallelismStr := strconv.Itoa(cmdArgs.Parallelism)
		args = args.AppendLoggableKV(parallelFlag, parallelismStr)
	}

	return stringSliceCommand(args)
}
