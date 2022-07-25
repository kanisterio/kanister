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

type ServerSetUserCommandArgs struct {
	*CommandArgs
	NewUsername  string
	UserPassword string
}

// ServerSetUser returns the kopia command setting password for existing user for the Kopia API Server
func ServerSetUser(
	serverSetUserArgs ServerSetUserCommandArgs) []string {
	args := commonArgs(serverSetUserArgs.EncryptionKey, serverSetUserArgs.ConfigFilePath, serverSetUserArgs.LogDirectory, false)
	args = args.AppendLoggable(serverSubCommand, userSubCommand, setSubCommand, serverSetUserArgs.NewUsername)
	args = args.AppendRedactedKV(userPasswordFlag, serverSetUserArgs.UserPassword)

	return stringSliceCommand(args)
}
