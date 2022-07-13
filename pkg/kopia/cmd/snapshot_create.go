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

	"github.com/kanisterio/kanister/pkg/kopia"
	"github.com/kanisterio/kanister/pkg/logsafe"
	"github.com/kanisterio/kanister/pkg/utils"
)

// SnapshotCreate returns the kopia command for creation of a snapshot
// TODO: Have better mechanism to apply global flags
func SnapshotCreate(encryptionKey, pathToBackup, configFilePath, logDirectory string) []string {
	return stringSliceCommand(snapshotCreate(encryptionKey, pathToBackup, configFilePath, logDirectory))
}

func snapshotCreate(encryptionKey, pathToBackup, configFilePath, logDirectory string) logsafe.Cmd {
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
