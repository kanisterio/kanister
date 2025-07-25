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
	"fmt"
	"strconv"
	"time"

	"github.com/kanisterio/kanister/pkg/logsafe"
	"github.com/kanisterio/kanister/pkg/utils"
)

const (
	manifestTypeSnapshotFilter = "type:snapshot"
)

type SnapshotCreateCommandArgs struct {
	*CommandArgs
	PathToBackup           string
	Tags                   []string
	ProgressUpdateInterval time.Duration
	Parallelism            int
	EstimationType         string
	EstimationThreshold    int
}

// SnapshotCreate returns the kopia command for creation of a snapshot
func SnapshotCreate(cmdArgs SnapshotCreateCommandArgs) []string {
	// Default to info log level unless specified otherwise.
	if cmdArgs.LogLevel == "" {
		// Make a copy of the common command args, set the log level to info.
		common := *cmdArgs.CommandArgs
		common.LogLevel = LogLevelInfo
		cmdArgs.CommandArgs = &common
	}

	parallelismStr := strconv.Itoa(cmdArgs.Parallelism)
	args := commonArgs(cmdArgs.CommandArgs)
	args = args.AppendLoggable(snapshotSubCommand, createSubCommand, cmdArgs.PathToBackup, jsonFlag)
	args = args.AppendLoggableKV(parallelFlag, parallelismStr)

	if cmdArgs.EstimationType != "" {
		args = args.AppendLoggableKV(progressEstimationTypeFlag, cmdArgs.EstimationType)
		if cmdArgs.EstimationType == progressEstimationTypeAdaptive {
			threshold := cmdArgs.EstimationThreshold
			if threshold == 0 {
				threshold = defaultAdaptiveEstimationThreshold
			}
			thresholdStr := strconv.Itoa(threshold)
			args = args.AppendLoggableKV(adaptiveEstimationThresholdFlag, thresholdStr)
		}
	}
	args = addTags(cmdArgs.Tags, args)

	// kube.Exec might timeout after 4h if there is no output from the command
	// Setting it to 1h by default, instead of 1000000h so that kopia logs progress once every hour
	// In some cases, the update interval is set by the caller
	duration := "1h"
	if cmdArgs.ProgressUpdateInterval > 0 {
		duration = utils.DurationToString(utils.RoundUpDuration(cmdArgs.ProgressUpdateInterval))
	}
	args = args.AppendLoggableKV(progressUpdateIntervalFlag, duration)
	return stringSliceCommand(args)
}

type SnapshotRestoreCommandArgs struct {
	*CommandArgs
	SnapID                 string
	TargetPath             string
	SparseRestore          bool
	IgnorePermissionErrors bool
	SkipExisting           bool
	DeleteExtra            bool
	Parallelism            int
}

// SnapshotRestore returns kopia command restoring snapshots with given snap ID
func SnapshotRestore(cmdArgs SnapshotRestoreCommandArgs) []string {
	args := commonArgs(cmdArgs.CommandArgs)
	args = args.AppendLoggable(snapshotSubCommand, restoreSubCommand, cmdArgs.SnapID, cmdArgs.TargetPath)
	if cmdArgs.Parallelism > 0 {
		parallelismStr := strconv.Itoa(cmdArgs.Parallelism)
		args = args.AppendLoggableKV(parallelFlag, parallelismStr)
	}
	if cmdArgs.IgnorePermissionErrors {
		args = args.AppendLoggable(ignorePermissionsError)
	} else {
		args = args.AppendLoggable(noIgnorePermissionsError)
	}
	args = appendOptionalSnapshotRestoreFlags(args, cmdArgs)

	return stringSliceCommand(args)
}

func appendOptionalSnapshotRestoreFlags(args logsafe.Cmd, cmdArgs SnapshotRestoreCommandArgs) logsafe.Cmd {
	if cmdArgs.SparseRestore {
		args = args.AppendLoggable(sparseFlag)
	}
	if cmdArgs.SkipExisting {
		args = args.AppendLoggable(skipExistingFlag)
	}
	if cmdArgs.DeleteExtra {
		args = args.AppendLoggable(deleteExtraFlag)
	}
	return args
}

type SnapshotDeleteCommandArgs struct {
	*CommandArgs
	SnapID string
}

// SnapshotDelete returns the kopia command for deleting a snapshot with given snapshot ID
func SnapshotDelete(cmdArgs SnapshotDeleteCommandArgs) []string {
	args := commonArgs(cmdArgs.CommandArgs)
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
	args := commonArgs(cmdArgs.CommandArgs)
	args = args.AppendLoggable(snapshotSubCommand, expireSubCommand, cmdArgs.RootID)
	if cmdArgs.MustDelete {
		args = args.AppendLoggable(deleteFlag)
	}

	return stringSliceCommand(args)
}

type SnapListAllCommandArgs struct {
	*CommandArgs
}

// SnapListAll returns the kopia command for listing all snapshots in the repository with their sizes
func SnapListAll(cmdArgs SnapListAllCommandArgs) []string {
	args := commonArgs(cmdArgs.CommandArgs)
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
	args := commonArgs(cmdArgs.CommandArgs)
	args = args.AppendLoggable(manifestSubCommand, listSubCommand, jsonFlag)
	args = args.AppendLoggableKV(filterFlag, manifestTypeSnapshotFilter)

	return stringSliceCommand(args)
}

type SnapListByTagsCommandArgs struct {
	*CommandArgs
	Tags []string
}

func SnapListByTags(cmdArgs SnapListByTagsCommandArgs) []string {
	args := commonArgs(cmdArgs.CommandArgs)
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

type SnapshotVerifyCommandArgs struct {
	*CommandArgs
	DirectoryID        []string
	FileID             []string
	Sources            []string
	SnapshotIDs        []string
	VerifyFilesPercent *float64
	Parallelism        *int
	FileQueueLength    *int
	FileParallelism    *int
	MaxErrors          *int
	GetJSONOutput      bool
}

// SnapshotVerify returns kopia command verifying snapshots with given snapshot IDs.
func SnapshotVerify(cmdArgs SnapshotVerifyCommandArgs) []string {
	args := commonArgs(cmdArgs.CommandArgs)
	args = args.AppendLoggable(snapshotSubCommand, verifySubCommand)

	if cmdArgs.VerifyFilesPercent != nil {
		args = args.AppendLoggableKV(verifyFilesPercentFlag, fmt.Sprintf("%v", *cmdArgs.VerifyFilesPercent))
	}

	if cmdArgs.MaxErrors != nil {
		args = args.AppendLoggableKV(maxErrorsFlag, strconv.Itoa(*cmdArgs.MaxErrors))
	}

	if cmdArgs.Parallelism != nil {
		parallelismStr := strconv.Itoa(*cmdArgs.Parallelism)
		args = args.AppendLoggableKV(parallelFlag, parallelismStr)
	}

	if cmdArgs.FileQueueLength != nil {
		args = args.AppendLoggableKV(fileQueueLengthFlag, strconv.Itoa(*cmdArgs.FileQueueLength))
	}

	if cmdArgs.FileParallelism != nil {
		args = args.AppendLoggableKV(fileParallelismFlag, strconv.Itoa(*cmdArgs.FileParallelism))
	}

	for _, dirID := range cmdArgs.DirectoryID {
		args = args.AppendLoggableKV(directoryIDFlag, dirID)
	}

	for _, fileID := range cmdArgs.FileID {
		args = args.AppendLoggableKV(fileIDFlag, fileID)
	}

	for _, source := range cmdArgs.Sources {
		args = args.AppendLoggableKV(sourcesFlag, source)
	}

	if cmdArgs.GetJSONOutput {
		args = args.AppendLoggable(jsonFlag)
	}

	for _, snapID := range cmdArgs.SnapshotIDs {
		args = args.AppendLoggable(snapID)
	}

	return stringSliceCommand(args)
}
