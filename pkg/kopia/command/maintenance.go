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

type MaintenanceInfoCommandArgs struct {
	*CommandArgs
	GetJsonOutput bool
}

// MaintenanceInfo returns the kopia command to get maintenance info
func MaintenanceInfo(cmdArgs MaintenanceInfoCommandArgs) []string {
	return stringSliceCommand(maintenanceInfoCommand(cmdArgs))
}

func maintenanceInfoCommand(cmdArgs MaintenanceInfoCommandArgs) logsafe.Cmd {
	args := commonArgs(cmdArgs.EncryptionKey, cmdArgs.ConfigFilePath, cmdArgs.LogDirectory, false)
	args = args.AppendLoggable(maintenanceSubCommand, infoSubCommand)
	if cmdArgs.GetJsonOutput {
		args = args.AppendLoggable(jsonFlag)
	}
	return args
}

type MaintenanceSetOwnerCommandArgs struct {
	*CommandArgs
	CustomOwner string
}

// MaintenanceSetOwner returns the kopia command for setting custom maintenance owner
func MaintenanceSetOwner(cmdArgs MaintenanceSetOwnerCommandArgs) []string {
	return stringSliceCommand(maintenanceSetOwnerCommand(cmdArgs))
}

func maintenanceSetOwnerCommand(cmdArgs MaintenanceSetOwnerCommandArgs) logsafe.Cmd {
	args := commonArgs(cmdArgs.EncryptionKey, cmdArgs.ConfigFilePath, cmdArgs.LogDirectory, false)
	args = args.AppendLoggable(maintenanceSubCommand, setSubCommand)
	args = args.AppendLoggableKV(ownerFlag, cmdArgs.CustomOwner)
	return args
}

type MaintenanceRunCommandArgs struct {
	*CommandArgs
}

// MaintenanceRun returns the kopia command to run manual maintenance
func MaintenanceRun(cmdArgs MaintenanceRunCommandArgs) []string {
	return stringSliceCommand(maintenanceRunCommand(cmdArgs))
}

func maintenanceRunCommand(cmdArgs MaintenanceRunCommandArgs) logsafe.Cmd {
	args := commonArgs(cmdArgs.EncryptionKey, cmdArgs.ConfigFilePath, cmdArgs.LogDirectory, false)
	args = args.AppendLoggable(maintenanceSubCommand, runSubCommand)
	return args
}
