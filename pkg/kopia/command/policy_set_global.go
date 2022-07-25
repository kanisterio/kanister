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

type PolicySetGlobalCommandArgs struct {
	*CommandArgs
	modifications policyChanges
}

// PolicySetGlobal returns the kopia command for modifying the global policy
func PolicySetGlobal(policySetGlobalArgs PolicySetGlobalCommandArgs) []string {
	args := commonArgs(policySetGlobalArgs.encryptionKey, policySetGlobalArgs.configFilePath, policySetGlobalArgs.logDirectory, false)
	args = args.AppendLoggable(policySubCommand, setSubCommand, globalFlag)
	for field, val := range policySetGlobalArgs.modifications {
		args = args.AppendLoggableKV(field, val)
	}

	return stringSliceCommand(args)
}
