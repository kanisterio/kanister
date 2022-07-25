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

type MaintenanceSetOwnerCommandArgs struct {
	*CommandArgs
	CustomOwner string
}

// MaintenanceSetOwner returns the kopia command for setting custom maintenance owner
func MaintenanceSetOwner(maintenanceSetOwnerArgs MaintenanceSetOwnerCommandArgs) []string {
	args := commonArgs(maintenanceSetOwnerArgs.EncryptionKey, maintenanceSetOwnerArgs.ConfigFilePath, maintenanceSetOwnerArgs.LogDirectory, false)
	args = args.AppendLoggable(maintenanceSubCommand, setSubCommand)
	args = args.AppendLoggableKV(ownerFlag, maintenanceSetOwnerArgs.CustomOwner)
	return stringSliceCommand(args)
}
