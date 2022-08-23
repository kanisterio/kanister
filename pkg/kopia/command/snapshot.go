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
	"strconv"

	"github.com/kanisterio/kanister/pkg/kopia"
	"github.com/kanisterio/kanister/pkg/utils"
)

const (
	// kube.Exec might timeout after 4h if there is no output from the command
	// Setting it to 1h instead of 1000000h so that kopia logs progress once every hour
	longUpdateInterval = "1h"

	requireLogLevelInfo = true
)

type SnapshotCreateCommandArgs struct {
	*CommandArgs
	PathToBackup string
}

// SnapshotCreate returns the kopia command for creation of a snapshot
// TODO: Have better mechanism to apply global flags
func SnapshotCreate(snapshotCreateArgs SnapshotCreateCommandArgs) []string {
	parallelismStr := strconv.Itoa(utils.GetEnvAsIntOrDefault(kopia.DataStoreParallelUploadVarName, kopia.DefaultDataStoreParallelUpload))
	args := commonArgs(snapshotCreateArgs.EncryptionKey, snapshotCreateArgs.ConfigFilePath, snapshotCreateArgs.LogDirectory, requireLogLevelInfo)
	args = args.AppendLoggable(snapshotSubCommand, createSubCommand, snapshotCreateArgs.PathToBackup, jsonFlag)
	args = args.AppendLoggableKV(parallelFlag, parallelismStr)
	args = args.AppendLoggableKV(progressUpdateIntervalFlag, longUpdateInterval)

	return stringSliceCommand(args)
}

type SnapshotRestoreCommandArgs struct {
	*CommandArgs
	SnapID        string
	TargetPath    string
	SparseRestore bool
}

// SnapshotRestore returns kopia command restoring snapshots with given snap ID
func SnapshotRestore(snapshotRestoreArgs SnapshotRestoreCommandArgs) []string {
	args := commonArgs(snapshotRestoreArgs.EncryptionKey, snapshotRestoreArgs.ConfigFilePath, snapshotRestoreArgs.LogDirectory, false)
	args = args.AppendLoggable(snapshotSubCommand, restoreSubCommand, snapshotRestoreArgs.SnapID, snapshotRestoreArgs.TargetPath)
	if snapshotRestoreArgs.SparseRestore {
		args = args.AppendLoggable(sparseFlag)
	}

	return stringSliceCommand(args)
}

type SnapshotDeleteCommandArgs struct {
	*CommandArgs
	SnapID string
}

// SnapshotDelete returns the kopia command for deleting a snapshot with given snapshot ID
func SnapshotDelete(snapshotDeleteArgs SnapshotDeleteCommandArgs) []string {
	args := commonArgs(snapshotDeleteArgs.EncryptionKey, snapshotDeleteArgs.ConfigFilePath, snapshotDeleteArgs.LogDirectory, false)
	args = args.AppendLoggable(snapshotSubCommand, deleteSubCommand, snapshotDeleteArgs.SnapID, unsafeIgnoreSourceFlag)

	return stringSliceCommand(args)
}

type SnapshotExpireCommandArgs struct {
	*CommandArgs
	RootID     string
	MustDelete bool
}

// SnapshotExpire returns the kopia command for removing snapshots with given root ID
func SnapshotExpire(snapshotExpireArgs SnapshotExpireCommandArgs) []string {
	args := commonArgs(snapshotExpireArgs.EncryptionKey, snapshotExpireArgs.ConfigFilePath, snapshotExpireArgs.LogDirectory, false)
	args = args.AppendLoggable(snapshotSubCommand, expireSubCommand, snapshotExpireArgs.RootID)
	if snapshotExpireArgs.MustDelete {
		args = args.AppendLoggable(deleteFlag)
	}

	return stringSliceCommand(args)
}

type SnapshotGCCommandArgs struct {
	*CommandArgs
}

// SnapshotGC returns the kopia command for issuing kopia snapshot gc
func SnapshotGC(snapshotGCArgs SnapshotGCCommandArgs) []string {
	args := commonArgs(snapshotGCArgs.EncryptionKey, snapshotGCArgs.ConfigFilePath, snapshotGCArgs.LogDirectory, false)
	args = args.AppendLoggable(snapshotSubCommand, gcSubCommand, deleteFlag)

	return stringSliceCommand(args)
}

type SnapListAllCommandArgs struct {
	*CommandArgs
}

// SnapListAll returns the kopia command for listing all snapshots in the repository with their sizes
func SnapListAll(snapListAllArgs SnapListAllCommandArgs) []string {
	args := commonArgs(snapListAllArgs.EncryptionKey, snapListAllArgs.ConfigFilePath, snapListAllArgs.LogDirectory, false)
	args = args.AppendLoggable(
		snapshotSubCommand,
		listSubCommand,
		allFlag,
		deltaFlag,
		showIdenticalFlag,
		jsonFlag,
	)

	return stringSliceCommand(args)
}

type SnapListAllWithSnapIDsCommandArgs struct {
	*CommandArgs
}

// SnapListAllWithSnapIDs returns the kopia command for listing all snapshots in the repository with snapshotIDs
func SnapListAllWithSnapIDs(snapListAllWithSnapIDsArgs SnapListAllWithSnapIDsCommandArgs) []string {
	args := commonArgs(snapListAllWithSnapIDsArgs.EncryptionKey, snapListAllWithSnapIDsArgs.ConfigFilePath, snapListAllWithSnapIDsArgs.LogDirectory, false)
	args = args.AppendLoggable(manifestSubCommand, listSubCommand, jsonFlag)
	args = args.AppendLoggableKV(filterFlag, kopia.ManifestTypeSnapshotFilter)

	return stringSliceCommand(args)
}
