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
	Tags         []string
}

// SnapshotCreate returns the kopia command for creation of a snapshot
func SnapshotCreate(cmdArgs SnapshotCreateCommandArgs) []string {
	parallelismStr := strconv.Itoa(utils.GetEnvAsIntOrDefault(kopia.DataStoreParallelUploadVarName, kopia.DefaultDataStoreParallelUpload))
	args := commonArgs(cmdArgs.CommandArgs, requireLogLevelInfo)
	args = args.AppendLoggable(snapshotSubCommand, createSubCommand, cmdArgs.PathToBackup, jsonFlag)
	args = args.AppendLoggableKV(parallelFlag, parallelismStr)
	args = args.AppendLoggableKV(progressUpdateIntervalFlag, longUpdateInterval)
	args = addTags(cmdArgs.Tags, args)
	return stringSliceCommand(args)
}

type SnapshotRestoreCommandArgs struct {
	*CommandArgs
	SnapID        string
	TargetPath    string
	SparseRestore bool
}

// SnapshotRestore returns kopia command restoring snapshots with given snap ID
func SnapshotRestore(cmdArgs SnapshotRestoreCommandArgs) []string {
	args := commonArgs(cmdArgs.CommandArgs, false)
	args = args.AppendLoggable(snapshotSubCommand, restoreSubCommand, cmdArgs.SnapID, cmdArgs.TargetPath)
	if cmdArgs.SparseRestore {
		args = args.AppendLoggable(sparseFlag)
	}

	return stringSliceCommand(args)
}

type SnapshotDeleteCommandArgs struct {
	*CommandArgs
	SnapID string
}

// SnapshotDelete returns the kopia command for deleting a snapshot with given snapshot ID
func SnapshotDelete(cmdArgs SnapshotDeleteCommandArgs) []string {
	args := commonArgs(cmdArgs.CommandArgs, false)
	args = args.AppendLoggable(snapshotSubCommand, deleteSubCommand, cmdArgs.SnapID, unsafeIgnoreSourceFlag)

	return stringSliceCommand(args)
}

type SnapshotExpireCommandArgs struct {
	*CommandArgs
	RootID     string
	MustDelete bool
}

// SnapshotExpire returns the kopia command for removing snapshots with given root ID
func SnapshotExpire(cmdArgs SnapshotExpireCommandArgs) []string {
	args := commonArgs(cmdArgs.CommandArgs, false)
	args = args.AppendLoggable(snapshotSubCommand, expireSubCommand, cmdArgs.RootID)
	if cmdArgs.MustDelete {
		args = args.AppendLoggable(deleteFlag)
	}

	return stringSliceCommand(args)
}

type SnapshotGCCommandArgs struct {
	*CommandArgs
}

// SnapshotGC returns the kopia command for issuing kopia snapshot gc
func SnapshotGC(cmdArgs SnapshotGCCommandArgs) []string {
	args := commonArgs(cmdArgs.CommandArgs, false)
	args = args.AppendLoggable(snapshotSubCommand, gcSubCommand, deleteFlag)

	return stringSliceCommand(args)
}

type SnapListAllCommandArgs struct {
	*CommandArgs
}

// SnapListAll returns the kopia command for listing all snapshots in the repository with their sizes
func SnapListAll(cmdArgs SnapListAllCommandArgs) []string {
	args := commonArgs(cmdArgs.CommandArgs, false)
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
func SnapListAllWithSnapIDs(cmdArgs SnapListAllWithSnapIDsCommandArgs) []string {
	args := commonArgs(cmdArgs.CommandArgs, false)
	args = args.AppendLoggable(manifestSubCommand, listSubCommand, jsonFlag)
	args = args.AppendLoggableKV(filterFlag, kopia.ManifestTypeSnapshotFilter)

	return stringSliceCommand(args)
}

type SnapListByTagsCommandArgs struct {
	*CommandArgs
	Tags []string
}

func SnapListByTags(cmdArgs SnapListByTagsCommandArgs) []string {
	args := commonArgs(cmdArgs.CommandArgs, false)
	args = args.AppendLoggable(
		snapshotSubCommand,
		listSubCommand,
		allFlag,
		deltaFlag,
		showIdenticalFlag,
		jsonFlag,
	)
	args = addTags(cmdArgs.Tags, args)
	return stringSliceCommand(args)
}
