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

import (
	"github.com/kanisterio/kanister/pkg/kopia/cli/args"
)

type ServerStartCommandArgs struct {
	*CommandArgs
	CacheArgs
	CacheDirectory       string
	ServerAddress        string
	TLSCertFile          string
	TLSKeyFile           string
	ServerUsername       string
	ServerPassword       string
	HtpasswdFilePath     string
	ReadOnly             bool
	AutoGenerateCert     bool
	Background           bool
	EnablePprof          bool
	MetricsListenAddress string
}

// ServerStart returns the kopia command for starting the Kopia API Server
func ServerStart(cmdArgs ServerStartCommandArgs) []string {
	args := commonArgs(&CommandArgs{ConfigFilePath: cmdArgs.ConfigFilePath, LogDirectory: cmdArgs.LogDirectory})

	if cmdArgs.AutoGenerateCert {
		args = args.AppendLoggable(serverSubCommand, startSubCommand, tlsGenerateCertFlag)
	} else {
		args = args.AppendLoggable(serverSubCommand, startSubCommand)
	}
	args = args.AppendLoggableKV(addressFlag, cmdArgs.ServerAddress)
	args = args.AppendLoggableKV(tlsCertFilePath, cmdArgs.TLSCertFile)
	args = args.AppendLoggableKV(tlsKeyFilePath, cmdArgs.TLSKeyFile)

	if cmdArgs.HtpasswdFilePath != "" {
		args = args.AppendLoggableKV(htpasswdFilePath, cmdArgs.HtpasswdFilePath)
	}

	args = args.AppendLoggableKV(serverUsernameFlag, cmdArgs.ServerUsername)
	args = args.AppendRedactedKV(serverPasswordFlag, cmdArgs.ServerPassword)

	args = args.AppendLoggableKV(serverControlUsernameFlag, cmdArgs.ServerUsername)
	args = args.AppendRedactedKV(serverControlPasswordFlag, cmdArgs.ServerPassword)

	args = cmdArgs.kopiaCacheArgs(args, cmdArgs.CacheDirectory)

	if cmdArgs.ReadOnly {
		args = args.AppendLoggable(readOnlyFlag)
	}

	if cmdArgs.EnablePprof {
		args = args.AppendLoggable(enablePprof)
	}

	if cmdArgs.MetricsListenAddress != "" {
		args = args.AppendLoggableKV(metricsListerAddress, cmdArgs.MetricsListenAddress)
	}

	if cmdArgs.Background {
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
func ServerRefresh(cmdArgs ServerRefreshCommandArgs) []string {
	args := commonArgs(cmdArgs.CommandArgs)
	args = args.AppendLoggable(serverSubCommand, refreshSubCommand)
	args = args.AppendRedactedKV(serverCertFingerprint, cmdArgs.Fingerprint)
	args = args.AppendLoggableKV(addressFlag, cmdArgs.ServerAddress)
	args = args.AppendLoggableKV(serverUsernameFlag, cmdArgs.ServerUsername)
	args = args.AppendRedactedKV(serverPasswordFlag, cmdArgs.ServerPassword)

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
func ServerStatus(cmdArgs ServerStatusCommandArgs) []string {
	args := commonArgs(&CommandArgs{ConfigFilePath: cmdArgs.ConfigFilePath, LogDirectory: cmdArgs.LogDirectory})
	args = args.AppendLoggable(serverSubCommand, statusSubCommand)
	args = args.AppendLoggableKV(addressFlag, cmdArgs.ServerAddress)
	args = args.AppendRedactedKV(serverCertFingerprint, cmdArgs.Fingerprint)
	args = args.AppendLoggableKV(serverUsernameFlag, cmdArgs.ServerUsername)
	args = args.AppendRedactedKV(serverPasswordFlag, cmdArgs.ServerPassword)

	return stringSliceCommand(args)
}

type ServerListUserCommmandArgs struct {
	*CommandArgs
}

// ServerListUser returns the kopia command to list users from the Kopia API Server
func ServerListUser(cmdArgs ServerListUserCommmandArgs) []string {
	args := commonArgs(cmdArgs.CommandArgs)
	args = args.AppendLoggable(serverSubCommand, userSubCommand, listSubCommand, jsonFlag)

	return stringSliceCommand(args)
}

type ServerSetUserCommandArgs struct {
	*CommandArgs
	NewUsername  string
	UserPassword string
}

// ServerSetUser returns the kopia command setting password for existing user for the Kopia API Server
func ServerSetUser(cmdArgs ServerSetUserCommandArgs) []string {
	command := commonArgs(cmdArgs.CommandArgs)
	command = command.AppendLoggable(serverSubCommand, userSubCommand, setSubCommand, cmdArgs.NewUsername)
	command = command.AppendRedactedKV(userPasswordFlag, cmdArgs.UserPassword)

	command = args.UserAddSet.AppendToCmd(command)

	return stringSliceCommand(command)
}

type ServerAddUserCommandArgs struct {
	*CommandArgs
	NewUsername  string
	UserPassword string
}

// ServerAddUser returns the kopia command adding a new user to the Kopia API Server
func ServerAddUser(cmdArgs ServerAddUserCommandArgs) []string {
	command := commonArgs(cmdArgs.CommandArgs)
	command = command.AppendLoggable(serverSubCommand, userSubCommand, addSubCommand, cmdArgs.NewUsername)
	command = command.AppendRedactedKV(userPasswordFlag, cmdArgs.UserPassword)

	command = args.UserAddSet.AppendToCmd(command)

	return stringSliceCommand(command)
}
