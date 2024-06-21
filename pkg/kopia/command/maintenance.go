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

type MaintenanceInfoCommandArgs struct {
	*CommandArgs
	GetJSONOutput bool
}

// MaintenanceInfo returns the kopia command to get maintenance info
func MaintenanceInfo(cmdArgs MaintenanceInfoCommandArgs) []string {
	args := commonArgs(cmdArgs.CommandArgs)
	args = args.AppendLoggable(maintenanceSubCommand, infoSubCommand)
	if cmdArgs.GetJSONOutput {
		args = args.AppendLoggable(jsonFlag)
	}

	return stringSliceCommand(args)
}

type MaintenanceSetOwnerCommandArgs struct {
	*CommandArgs
	CustomOwner string
}

// MaintenanceSetOwner returns the kopia command for setting custom maintenance owner
func MaintenanceSetOwner(cmdArgs MaintenanceSetOwnerCommandArgs) []string {
	args := commonArgs(cmdArgs.CommandArgs)
	args = args.AppendLoggable(maintenanceSubCommand, setSubCommand)
	args = args.AppendLoggableKV(ownerFlag, cmdArgs.CustomOwner)
	return stringSliceCommand(args)
}

type MaintenanceRunCommandArgs struct {
	*CommandArgs
}

// MaintenanceRunCommand returns the kopia command to run manual maintenance
func MaintenanceRunCommand(cmdArgs MaintenanceRunCommandArgs) []string {
	args := commonArgs(cmdArgs.CommandArgs)
	args = args.AppendLoggable(maintenanceSubCommand, runSubCommand)

	return stringSliceCommand(args)
}
