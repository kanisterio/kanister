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

type ServerStatusCommandArgs struct {
	*CommandArgs
	ServerAddress  string
	ServerUsername string
	ServerPassword string
	Fingerprint    string
}

// ServerStatus returns the kopia command for checking status of the Kopia API Server
func ServerStatus(serverStatusArgs ServerStatusCommandArgs) []string {
	args := commonArgs("", serverStatusArgs.ConfigFilePath, serverStatusArgs.LogDirectory, false)
	args = args.AppendLoggable(serverSubCommand, statusSubCommand)
	args = args.AppendLoggableKV(addressFlag, serverStatusArgs.ServerAddress)
	args = args.AppendRedactedKV(serverCertFingerprint, serverStatusArgs.Fingerprint)
	args = args.AppendLoggableKV(serverUsernameFlag, serverStatusArgs.ServerUsername)
	args = args.AppendRedactedKV(serverPasswordFlag, serverStatusArgs.ServerPassword)

	return stringSliceCommand(args)
}
