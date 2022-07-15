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

package cmd

import (
	"strconv"

	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/kopia"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/logsafe"
	"github.com/kanisterio/kanister/pkg/utils"
)

func bashCommand(args logsafe.Cmd) []string {
	log.Info().Print("Kopia Command", field.M{"Command": args.String()})
	return []string{"bash", "-o", "errexit", "-c", args.PlainText()}
}

func stringSliceCommand(args logsafe.Cmd) []string {
	log.Info().Print("Kopia Command", field.M{"Command": args.String()})
	return args.StringSliceCMD()
}

func kopiaArgs(password, configFilePath, logDirectory string, requireInfoLevel bool) logsafe.Cmd {
	c := logsafe.NewLoggable(kopiaCommand)
	if requireInfoLevel {
		c = c.AppendLoggable(logLevelInfoFlag)
	} else {
		c = c.AppendLoggable(logLevelErrorFlag)
	}
	if configFilePath != "" {
		c = c.AppendLoggableKV(configFileFlag, configFilePath)
	}
	if logDirectory != "" {
		c = c.AppendLoggableKV(logDirectoryFlag, logDirectory)
	}
	if password != "" {
		c = c.AppendRedactedKV(passwordFlag, password)
	}
	return c
}

// ExecKopiaArgs returns the basic Argv for executing kopia with the given config file path.
func ExecKopiaArgs(configFilePath string) []string {
	return kopiaArgs("", configFilePath, "", false).StringSliceCMD()
}

// SnapshotCreateCommand returns the kopia command for creation of a snapshot
// TODO: Have better mechanism to apply global flags
func SnapshotCreateCommand(encryptionKey, pathToBackup, configFilePath, logDirectory string) []string {
	return stringSliceCommand(snapshotCreateCommand(encryptionKey, pathToBackup, configFilePath, logDirectory))
}

func snapshotCreateCommand(encryptionKey, pathToBackup, configFilePath, logDirectory string) logsafe.Cmd {
	const (
		// kube.Exec might timeout after 4h if there is no output from the command
		// Setting it to 1h instead of 1000000h so that kopia logs progress once every hour
		longUpdateInterval = "1h"

		requireLogLevelInfo = true
	)

	parallelismStr := strconv.Itoa(utils.GetEnvAsIntOrDefault(kopia.DataStoreParallelUploadVarName, kopia.DefaultDataStoreParallelUpload))
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, requireLogLevelInfo)
	args = args.AppendLoggable(snapshotSubCommand, createSubCommand, pathToBackup, jsonFlag)
	args = args.AppendLoggableKV(parallelFlag, parallelismStr)
	args = args.AppendLoggableKV(progressUpdateIntervalFlag, longUpdateInterval)

	return args
}

// SnapshotExpireCommand returns the kopia command for removing snapshots with given root ID
func SnapshotExpireCommand(encryptionKey, rootID, configFilePath, logDirectory string, mustDelete bool) []string {
	return stringSliceCommand(snapshotExpireCommand(encryptionKey, rootID, configFilePath, logDirectory, mustDelete))
}

func snapshotExpireCommand(encryptionKey, rootID, configFilePath, logDirectory string, mustDelete bool) logsafe.Cmd {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(snapshotSubCommand, expireSubCommand, rootID)
	if mustDelete {
		args = args.AppendLoggable(deleteFlag)
	}

	return args
}

// SnapshotRestoreCommand returns kopia command restoring snapshots with given snap ID
func SnapshotRestoreCommand(encryptionKey, snapID, targetPath, configFilePath, logDirectory string, sparseRestore bool) []string {
	return stringSliceCommand(snapshotRestoreCommand(encryptionKey, snapID, targetPath, configFilePath, logDirectory, sparseRestore))
}

func snapshotRestoreCommand(encryptionKey, snapID, targetPath, configFilePath, logDirectory string, sparseRestore bool) logsafe.Cmd {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(snapshotSubCommand, restoreSubCommand, snapID, targetPath)
	if sparseRestore {
		args = args.AppendLoggable(sparseFlag)
	}

	return args
}

// RestoreCommand returns the kopia command for restoring root of a snapshot with given root ID
func RestoreCommand(encryptionKey, rootID, targetPath, configFilePath, logDirectory string) []string {
	return stringSliceCommand(restoreCommand(encryptionKey, rootID, targetPath, configFilePath, logDirectory))
}

func restoreCommand(encryptionKey, rootID, targetPath, configFilePath, logDirectory string) logsafe.Cmd {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(restoreSubCommand, rootID, targetPath)

	return args
}

// DeleteCommand returns the kopia command for deleting a snapshot with given snapshot ID
func DeleteCommand(encryptionKey, snapID, configFilePath, logDirectory string) []string {
	return stringSliceCommand(deleteCommand(encryptionKey, snapID, configFilePath, logDirectory))
}

func deleteCommand(encryptionKey, snapID, configFilePath, logDirectory string) logsafe.Cmd {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(snapshotSubCommand, deleteSubCommand, snapID, unsafeIgnoreSourceFlag)

	return args
}

// SnapshotGCCommand returns the kopia command for issuing kopia snapshot gc
func SnapshotGCCommand(encryptionKey, configFilePath, logDirectory string) []string {
	return stringSliceCommand(snapshotGCCommand(encryptionKey, configFilePath, logDirectory))
}

func snapshotGCCommand(encryptionKey, configFilePath, logDirectory string) logsafe.Cmd {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(snapshotSubCommand, gcSubCommand, deleteFlag)

	return args
}

// MaintenanceSetCommandWithOwner returns the kopia command for setting custom maintenance owner
func MaintenanceSetCommandWithOwner(encryptionKey, configFilePath, logDirectory, customOwner string) []string {
	return stringSliceCommand(maintenanceSetOwner(encryptionKey, configFilePath, logDirectory, customOwner))
}

func maintenanceSetOwner(encryptionKey, configFilePath, logDirectory, customOwner string) logsafe.Cmd {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(maintenanceSubCommand, setSubCommand)
	args = args.AppendLoggableKV(ownerFlag, customOwner)
	return args
}

// MaintenanceRunCommand returns the kopia command to run manual maintenance
func MaintenanceRunCommand(encryptionKey, configFilePath, logDirectory string) []string {
	return stringSliceCommand(maintenanceRunCommand(encryptionKey, configFilePath, logDirectory))
}

func maintenanceRunCommand(encryptionKey, configFilePath, logDirectory string) logsafe.Cmd {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(maintenanceSubCommand, runSubCommand)

	return args
}

// MaintenanceInfoCommand returns the kopia command to get maintenance info
func MaintenanceInfoCommand(encryptionKey, configFilePath, logDirectory string, getJsonOutput bool) []string {
	return stringSliceCommand(maintenanceInfoCommand(encryptionKey, configFilePath, logDirectory, getJsonOutput))
}

func maintenanceInfoCommand(encryptionKey, configFilePath, logDirectory string, getJsonOutput bool) logsafe.Cmd {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(maintenanceSubCommand, infoSubCommand)
	if getJsonOutput {
		args = args.AppendLoggable(jsonFlag)
	}

	return args
}

// BlobList returns the kopia command for listing blobs in the repository with their sizes
func BlobList(encryptionKey, configFilePath, logDirectory string) []string {
	return stringSliceCommand(blobList(encryptionKey, configFilePath, logDirectory))
}

func blobList(encryptionKey, configFilePath, logDirectory string) logsafe.Cmd {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(blobSubCommand, listSubCommand)

	return args
}

// BlobStats returns the kopia command to get the blob stats
func BlobStats(encryptionKey, configFilePath, logDirectory string) []string {
	return stringSliceCommand(blobStats(encryptionKey, configFilePath, logDirectory))
}

func blobStats(encryptionKey, configFilePath, logDirectory string) logsafe.Cmd {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(blobSubCommand, statsSubCommand, rawFlag)

	return args
}

// SnapListAll returns the kopia command for listing all snapshots in the repository with their sizes
func SnapListAll(encryptionKey, configFilePath, logDirectory string) []string {
	return stringSliceCommand(snapListAll(encryptionKey, configFilePath, logDirectory))
}

func snapListAll(encryptionKey, configFilePath, logDirectory string) logsafe.Cmd {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(
		snapshotSubCommand,
		listSubCommand,
		allFlag,
		deltaFlag,
		showIdenticalFlag,
		jsonFlag,
	)

	return args
}

// SnapListAllWithSnapIDs returns the kopia command for listing all snapshots in the repository with snapshotIDs
func SnapListAllWithSnapIDs(encryptionKey, configFilePath, logDirectory string) []string {
	return stringSliceCommand(snapListAllWithSnapIDs(encryptionKey, configFilePath, logDirectory))
}

func snapListAllWithSnapIDs(encryptionKey, configFilePath, logDirectory string) logsafe.Cmd {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(manifestSubCommand, listSubCommand, jsonFlag)
	args = args.AppendLoggableKV(filterFlag, kopia.ManifestTypeSnapshotFilter)

	return args
}

// ServerStartCommand returns the kopia command for starting the Kopia API Server
func ServerStartCommand(
	configFilePath,
	logDirectory,
	serverAddress,
	tlsCertFile,
	tlsKeyFile,
	serverUsername,
	serverPassword string,
	autoGenerateCert,
	background bool,
) []string {
	return bashCommand(serverStartCommand(
		configFilePath,
		logDirectory,
		serverAddress,
		tlsCertFile,
		tlsKeyFile,
		serverUsername,
		serverPassword,
		autoGenerateCert,
		background,
	))
}

func serverStartCommand(
	configFilePath,
	logDirectory,
	serverAddress,
	tlsCertFile,
	tlsKeyFile,
	serverUsername,
	serverPassword string,
	autoGenerateCert,
	background bool,
) logsafe.Cmd {
	args := kopiaArgs("", configFilePath, logDirectory, false)

	if autoGenerateCert {
		args = args.AppendLoggable(serverSubCommand, startSubCommand, tlsGenerateCertFlag)
	} else {
		args = args.AppendLoggable(serverSubCommand, startSubCommand)
	}
	args = args.AppendLoggableKV(addressFlag, serverAddress)
	args = args.AppendLoggableKV(tlsCertFilePath, tlsCertFile)
	args = args.AppendLoggableKV(tlsKeyFilePath, tlsKeyFile)
	args = args.AppendLoggableKV(serverUsernameFlag, serverUsername)
	args = args.AppendRedactedKV(serverPasswordFlag, serverPassword)

	args = args.AppendLoggableKV(serverControlUsernameFlag, serverUsername)
	args = args.AppendRedactedKV(serverControlPasswordFlag, serverPassword)

	// TODO: Remove when GRPC support is added
	args = args.AppendLoggable(noGrpcFlag)

	if background {
		// To start the server and run in the background
		args = args.AppendLoggable(redirectToDevNull, runInBackground)
	}

	return args
}

// ServerStatusCommand returns the kopia command for checking status of the Kopia API Server
func ServerStatusCommand(
	configFilePath,
	logDirectory,
	serverAddress,
	serverUsername,
	serverPassword,
	fingerprint string,
) []string {
	return stringSliceCommand(serverStatusCommand(
		configFilePath,
		logDirectory,
		serverAddress,
		serverUsername,
		serverPassword,
		fingerprint,
	))
}

func serverStatusCommand(
	configFilePath,
	logDirectory,
	serverAddress,
	serverUsername,
	serverPassword,
	fingerprint string,
) logsafe.Cmd {
	args := kopiaArgs("", configFilePath, logDirectory, false)
	args = args.AppendLoggable(serverSubCommand, statusSubCommand)
	args = args.AppendLoggableKV(addressFlag, serverAddress)
	args = args.AppendRedactedKV(serverCertFingerprint, fingerprint)
	args = args.AppendLoggableKV(serverUsernameFlag, serverUsername)
	args = args.AppendRedactedKV(serverPasswordFlag, serverPassword)

	return args
}

// ServerAddUserCommand returns the kopia command adding a new user to the Kopia API Server
func ServerAddUserCommand(
	encryptionKey,
	configFilePath,
	logDirectory,
	newUsername,
	userPassword string,
) []string {
	return stringSliceCommand(serverAddUserCommand(
		encryptionKey,
		configFilePath,
		logDirectory,
		newUsername,
		userPassword,
	))
}

func serverAddUserCommand(
	encryptionKey,
	configFilePath,
	logDirectory,
	newUsername,
	userPassword string,
) logsafe.Cmd {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(serverSubCommand, userSubCommand, addSubCommand, newUsername)
	args = args.AppendRedactedKV(userPasswordFlag, userPassword)

	return args
}

// ServerSetUserCommand returns the kopia command setting password for existing user for the Kopia API Server
func ServerSetUserCommand(
	encryptionKey,
	configFilePath,
	logDirectory,
	newUsername,
	userPassword string,
) []string {
	return stringSliceCommand(serverSetUserCommand(
		encryptionKey,
		configFilePath,
		logDirectory,
		newUsername,
		userPassword,
	))
}

func serverSetUserCommand(
	encryptionKey,
	configFilePath,
	logDirectory,
	newUsername,
	userPassword string,
) logsafe.Cmd {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(serverSubCommand, userSubCommand, setSubCommand, newUsername)
	args = args.AppendRedactedKV(userPasswordFlag, userPassword)

	return args
}

// ServerListUserCommand returns the kopia command to list users from the Kopia API Server
func ServerListUserCommand(
	encryptionKey,
	configFilePath,
	logDirectory string,
) []string {
	return stringSliceCommand(serverListUserCommand(
		encryptionKey,
		configFilePath,
		logDirectory,
	))
}

func serverListUserCommand(
	encryptionKey,
	configFilePath,
	logDirectory string,
) logsafe.Cmd {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(serverSubCommand, userSubCommand, listSubCommand, jsonFlag)

	return args
}

// ServerRefreshCommand returns the kopia command for refreshing the Kopia API Server
// This helps allow new users to be able to connect to the Server instead of waiting for auto-refresh
func ServerRefreshCommand(
	encryptionKey,
	configFilePath,
	logDirectory,
	serverAddress,
	serverUsername,
	serverPassword,
	fingerprint string,
) []string {
	return stringSliceCommand(serverRefreshCommand(
		encryptionKey,
		configFilePath,
		logDirectory,
		serverAddress,
		serverUsername,
		serverPassword,
		fingerprint,
	))
}

func serverRefreshCommand(
	encryptionKey,
	configFilePath,
	logDirectory,
	serverAddress,
	serverUsername,
	serverPassword,
	fingerprint string,
) logsafe.Cmd {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(serverSubCommand, refreshSubCommand)
	args = args.AppendRedactedKV(serverCertFingerprint, fingerprint)
	args = args.AppendLoggableKV(addressFlag, serverAddress)
	args = args.AppendLoggableKV(serverUsernameFlag, serverUsername)
	args = args.AppendRedactedKV(serverPasswordFlag, serverPassword)

	return args
}

// policySetGlobalCommand returns the kopia command for modifying the global policy
func policySetGlobalCommand(encryptionKey, configFilePath, logDirectory string, modifications policyChanges) []string {
	return stringSliceCommand(policySetGlobalCommandSetup(encryptionKey, configFilePath, logDirectory, modifications))
}

func policySetGlobalCommandSetup(encryptionKey, configFilePath, logDirectory string, modifications policyChanges) logsafe.Cmd {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(policySubCommand, setSubCommand, globalFlag)
	for field, val := range modifications {
		args = args.AppendLoggableKV(field, val)
	}

	return args
}
