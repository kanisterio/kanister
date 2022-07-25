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

type ServerAddUserCommandArgs struct {
	*CommandArgs
	newUsername  string
	userPassword string
}

// ServerAddUser returns the kopia command adding a new user to the Kopia API Server
func ServerAddUser(serverAddUserArgs ServerAddUserCommandArgs) []string {
	args := commonArgs(serverAddUserArgs.encryptionKey, serverAddUserArgs.configFilePath, serverAddUserArgs.logDirectory, false)
	args = args.AppendLoggable(serverSubCommand, userSubCommand, addSubCommand, serverAddUserArgs.newUsername)
	args = args.AppendRedactedKV(userPasswordFlag, serverAddUserArgs.userPassword)

	return stringSliceCommand(args)
}
