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
	"time"

	"github.com/kanisterio/kanister/pkg/kopia"
	"github.com/kanisterio/kanister/pkg/logsafe"
	"github.com/kanisterio/kanister/pkg/utils"
)

const (
	// kube.Exec might timeout after 4h if there is no output from the command
	// Setting it to 1h instead of 1000000h so that kopia logs progress once every hour
	logUpdateInterval = "1h"

	requireLogLevelInfo = true
)

type SnapshotCreateCommandArgs struct {
	*CommandArgs
	PathToBackup           string
	ProgressUpdateInterval time.Duration
}

// SnapshotCreate returns the kopia command for creation of a snapshot
func SnapshotCreate(cmdArgs SnapshotCreateCommandArgs) []string {
	return stringSliceCommand(snapshotCreateCommand(cmdArgs))
}

func snapshotCreateCommand(cmdArgs SnapshotCreateCommandArgs) logsafe.Cmd {
	parallelismStr := strconv.Itoa(utils.GetEnvAsIntOrDefault(kopia.DataStoreParallelUploadVarName, kopia.DefaultDataStoreParallelUpload))
	args := commonArgs(cmdArgs.EncryptionKey, cmdArgs.ConfigFilePath, cmdArgs.LogDirectory, requireLogLevelInfo)
	args = args.AppendLoggable(snapshotSubCommand, createSubCommand, cmdArgs.PathToBackup, jsonFlag)
	args = args.AppendLoggableKV(parallelFlag, parallelismStr)

	// In some cases, the update interval is set by the caller
	duration := logUpdateInterval
	if cmdArgs.ProgressUpdateInterval > 0 {
		duration = utils.DurationToString(utils.RoundUpDuration(cmdArgs.ProgressUpdateInterval))
	}
	args = args.AppendLoggableKV(progressUpdateIntervalFlag, duration)

	return args
}

type SnapshotRestoreCommandArgs struct {
	*CommandArgs
	SnapID        string
	TargetPath    string
	SparseRestore bool
}

// SnapshotRestore returns kopia command restoring snapshots with given snap ID
func SnapshotRestore(cmdArgs SnapshotRestoreCommandArgs) []string {
	return stringSliceCommand(snapshotRestoreCommand(cmdArgs))
}

func snapshotRestoreCommand(cmdArgs SnapshotRestoreCommandArgs) logsafe.Cmd {
	args := commonArgs(cmdArgs.EncryptionKey, cmdArgs.ConfigFilePath, cmdArgs.LogDirectory, false)
	args = args.AppendLoggable(snapshotSubCommand, restoreSubCommand, cmdArgs.SnapID, cmdArgs.TargetPath)
	if cmdArgs.SparseRestore {
		args = args.AppendLoggable(sparseFlag)
	}
	return args
}

type SnapshotDeleteCommandArgs struct {
	*CommandArgs
	SnapID string
}

// SnapshotDelete returns the kopia command for deleting a snapshot with given snapshot ID
func SnapshotDelete(cmdArgs SnapshotDeleteCommandArgs) []string {
	return stringSliceCommand(snapshotDeleteCommand(cmdArgs))
}

func snapshotDeleteCommand(cmdArgs SnapshotDeleteCommandArgs) logsafe.Cmd {
	args := commonArgs(cmdArgs.EncryptionKey, cmdArgs.ConfigFilePath, cmdArgs.LogDirectory, false)
	args = args.AppendLoggable(snapshotSubCommand, deleteSubCommand, cmdArgs.SnapID, unsafeIgnoreSourceFlag)
	return args
}

type SnapshotExpireCommandArgs struct {
	*CommandArgs
	RootID     string
	MustDelete bool
}

// SnapshotExpire returns the kopia command for removing snapshots with given root ID
func SnapshotExpire(cmdArgs SnapshotExpireCommandArgs) []string {
	return stringSliceCommand(snapshotExpireCommand(cmdArgs))
}

func snapshotExpireCommand(cmdArgs SnapshotExpireCommandArgs) logsafe.Cmd {
	args := commonArgs(cmdArgs.EncryptionKey, cmdArgs.ConfigFilePath, cmdArgs.LogDirectory, false)
	args = args.AppendLoggable(snapshotSubCommand, expireSubCommand, cmdArgs.RootID)
	if cmdArgs.MustDelete {
		args = args.AppendLoggable(deleteFlag)
	}
	return args
}

type SnapshotGCCommandArgs struct {
	*CommandArgs
}

// SnapshotGC returns the kopia command for issuing kopia snapshot gc
func SnapshotGC(cmdArgs SnapshotGCCommandArgs) []string {
	return stringSliceCommand(snapshotGCCommand(cmdArgs))
}

func snapshotGCCommand(cmdArgs SnapshotGCCommandArgs) logsafe.Cmd {
	args := commonArgs(cmdArgs.EncryptionKey, cmdArgs.ConfigFilePath, cmdArgs.LogDirectory, false)
	args = args.AppendLoggable(snapshotSubCommand, gcSubCommand, deleteFlag)
	return args
}

type SnapListAllCommandArgs struct {
	*CommandArgs
}

// SnapListAll returns the kopia command for listing all snapshots in the repository with their sizes
func SnapListAll(cmdArgs SnapListAllCommandArgs) []string {
	return stringSliceCommand(snapListAllCommand(cmdArgs))
}

func snapListAllCommand(cmdArgs SnapListAllCommandArgs) logsafe.Cmd {
	args := commonArgs(cmdArgs.EncryptionKey, cmdArgs.ConfigFilePath, cmdArgs.LogDirectory, false)
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

type SnapListAllWithSnapIDsCommandArgs struct {
	*CommandArgs
}

// SnapListAllWithSnapIDs returns the kopia command for listing all snapshots in the repository with snapshotIDs
func SnapListAllWithSnapIDs(cmdArgs SnapListAllWithSnapIDsCommandArgs) []string {
	return stringSliceCommand(snapListAllWithSnapIDsCommand(cmdArgs))
}

func snapListAllWithSnapIDsCommand(cmdArgs SnapListAllWithSnapIDsCommandArgs) logsafe.Cmd {
	args := commonArgs(cmdArgs.EncryptionKey, cmdArgs.ConfigFilePath, cmdArgs.LogDirectory, false)
	args = args.AppendLoggable(manifestSubCommand, listSubCommand, jsonFlag)
	args = args.AppendLoggableKV(filterFlag, kopia.ManifestTypeSnapshotFilter)
	return args
}
