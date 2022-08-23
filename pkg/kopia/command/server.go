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

type ServerStartCommandArgs struct {
	*CommandArgs
	ServerAddress    string
	TLSCertFile      string
	TLSKeyFile       string
	ServerUsername   string
	ServerPassword   string
	AutoGenerateCert bool
	Background       bool
}

// ServerStart returns the kopia command for starting the Kopia API Server
func ServerStart(serverStartArgs ServerStartCommandArgs) []string {
	args := commonArgs("", serverStartArgs.ConfigFilePath, serverStartArgs.LogDirectory, false)

	if serverStartArgs.AutoGenerateCert {
		args = args.AppendLoggable(serverSubCommand, startSubCommand, tlsGenerateCertFlag)
	} else {
		args = args.AppendLoggable(serverSubCommand, startSubCommand)
	}
	args = args.AppendLoggableKV(addressFlag, serverStartArgs.ServerAddress)
	args = args.AppendLoggableKV(tlsCertFilePath, serverStartArgs.TLSCertFile)
	args = args.AppendLoggableKV(tlsKeyFilePath, serverStartArgs.TLSKeyFile)
	args = args.AppendLoggableKV(serverUsernameFlag, serverStartArgs.ServerUsername)
	args = args.AppendRedactedKV(serverPasswordFlag, serverStartArgs.ServerPassword)

	args = args.AppendLoggableKV(serverControlUsernameFlag, serverStartArgs.ServerUsername)
	args = args.AppendRedactedKV(serverControlPasswordFlag, serverStartArgs.ServerPassword)

	// TODO: Remove when GRPC support is added
	args = args.AppendLoggable(noGrpcFlag)

	if serverStartArgs.Background {
		// To start the server and run in the background
		args = args.AppendLoggable(redirectToDevNull, runInBackground)
	}

	return bashCommand(args)
}

type ServerRefreshCommandArgs struct {
	*CommandArgs
	ServerAddress  string
	ServerUsername string
	ServerPassword string
	Fingerprint    string
}

// ServerRefresh returns the kopia command for refreshing the Kopia API Server
// This helps allow new users to be able to connect to the Server instead of waiting for auto-refresh
func ServerRefresh(serverRefreshArgs ServerRefreshCommandArgs) []string {
	args := commonArgs(serverRefreshArgs.EncryptionKey, serverRefreshArgs.ConfigFilePath, serverRefreshArgs.LogDirectory, false)
	args = args.AppendLoggable(serverSubCommand, refreshSubCommand)
	args = args.AppendRedactedKV(serverCertFingerprint, serverRefreshArgs.Fingerprint)
	args = args.AppendLoggableKV(addressFlag, serverRefreshArgs.ServerAddress)
	args = args.AppendLoggableKV(serverUsernameFlag, serverRefreshArgs.ServerUsername)
	args = args.AppendRedactedKV(serverPasswordFlag, serverRefreshArgs.ServerPassword)

	return stringSliceCommand(args)
}

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

type ServerListUserCommmandArgs struct {
	*CommandArgs
}

// ServerListUser returns the kopia command to list users from the Kopia API Server
func ServerListUser(serverListUserArgs ServerListUserCommmandArgs) []string {
	args := commonArgs(serverListUserArgs.EncryptionKey, serverListUserArgs.ConfigFilePath, serverListUserArgs.LogDirectory, false)
	args = args.AppendLoggable(serverSubCommand, userSubCommand, listSubCommand, jsonFlag)

	return stringSliceCommand(args)
}

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

type ServerAddUserCommandArgs struct {
	*CommandArgs
	NewUsername  string
	UserPassword string
}

// ServerAddUser returns the kopia command adding a new user to the Kopia API Server
func ServerAddUser(serverAddUserArgs ServerAddUserCommandArgs) []string {
	args := commonArgs(serverAddUserArgs.EncryptionKey, serverAddUserArgs.ConfigFilePath, serverAddUserArgs.LogDirectory, false)
	args = args.AppendLoggable(serverSubCommand, userSubCommand, addSubCommand, serverAddUserArgs.NewUsername)
	args = args.AppendRedactedKV(userPasswordFlag, serverAddUserArgs.UserPassword)

	return stringSliceCommand(args)
}
