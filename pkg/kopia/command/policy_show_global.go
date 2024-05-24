// Copyright 2023 The Kanister Authors.
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

type PolicyShowGlobalCommandArgs struct {
	*CommandArgs
	GetJSONOutput bool
}

// PolicyShowGlobal returns the kopia command for showing the global policy.
func PolicyShowGlobal(cmdArgs PolicyShowGlobalCommandArgs) []string {
	args := commonArgs(cmdArgs.CommandArgs)
	args = args.AppendLoggable(policySubCommand, showSubCommand, globalFlag)
	if cmdArgs.GetJSONOutput {
		args = args.AppendLoggable(jsonFlag)
	}

	return stringSliceCommand(args)
}
