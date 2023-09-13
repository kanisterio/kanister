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
	"github.com/kanisterio/kanister/pkg/logsafe"
	"github.com/kanisterio/kanister/pkg/utils"
)

const (
	// dataStoreGeneralContentCacheSizeMBVarName is the name of the environment variable that controls
	// kopia content cache size for general command workloads
	dataStoreGeneralContentCacheSizeMBVarName = "DATA_STORE_GENERAL_CONTENT_CACHE_SIZE_MB"
	// defaultDataStoreGeneralMetadataCacheSizeMB is the default metadata cache size for general command workloads
	defaultDataStoreGeneralMetadataCacheSizeMB = 500
	// dataStoreGeneralMetadataCacheSizeMBVarName is the name of the environment variable that controls
	// kopia metadata cache size for general command workloads
	dataStoreGeneralMetadataCacheSizeMBVarName = "DATA_STORE_GENERAL_METADATA_CACHE_SIZE_MB"
	// defaultDataStoreRestoreContentCacheSizeMB is the default content cache size for restore workloads
	defaultDataStoreRestoreContentCacheSizeMB = 500
	// defaultDataStoreGeneralContentCacheSizeMB is the default content cache size for general command workloads
	defaultDataStoreGeneralContentCacheSizeMB = 0
	// dataStoreRestoreContentCacheSizeMBVarName is the name of the environment variable that controls
	// kopia content cache size for restore workloads
	dataStoreRestoreContentCacheSizeMBVarName = "DATA_STORE_RESTORE_CONTENT_CACHE_SIZE_MB"
	// defaultDataStoreRestoreMetadataCacheSizeMB is the default metadata cache size for restore workloads
	defaultDataStoreRestoreMetadataCacheSizeMB = 500
	// dataStoreRestoreMetadataCacheSizeMBVarName is the name of the environment variable that controls
	// kopia metadata cache size for restore workloads
	dataStoreRestoreMetadataCacheSizeMBVarName = "DATA_STORE_RESTORE_METADATA_CACHE_SIZE_MB"
)

type policyChanges map[string]string

// GetCacheSizeSettingsForSnapshot returns the feature setting cache size values to be used
// for initializing repositories that will be performing general command workloads that benefit from
// cacheing metadata only.
func GetCacheSizeSettingsForSnapshot() (contentCacheMB, metadataCacheMB int) {
	return utils.GetEnvAsIntOrDefault(dataStoreGeneralContentCacheSizeMBVarName, defaultDataStoreGeneralContentCacheSizeMB),
		utils.GetEnvAsIntOrDefault(dataStoreGeneralMetadataCacheSizeMBVarName, defaultDataStoreGeneralMetadataCacheSizeMB)
}

// GetCacheSizeSettingsForRestore returns the feature setting cache size values to be used
// for initializing repositories that will be performing restore workloads
func GetCacheSizeSettingsForRestore() (contentCacheMB, metadataCacheMB int) {
	return utils.GetEnvAsIntOrDefault(dataStoreRestoreContentCacheSizeMBVarName, defaultDataStoreRestoreContentCacheSizeMB),
		utils.GetEnvAsIntOrDefault(dataStoreRestoreMetadataCacheSizeMBVarName, defaultDataStoreRestoreMetadataCacheSizeMB)
}

// GetGeneralCacheSizeSettings returns the feature setting cache size values to be used
// for initializing repositories that will be performing general command workloads that benefit from
// cacheing metadata only.
func GetGeneralCacheSizeSettings() (contentCacheMB, metadataCacheMB int) {
	return utils.GetEnvAsIntOrDefault(dataStoreGeneralContentCacheSizeMBVarName, defaultDataStoreGeneralContentCacheSizeMB),
		utils.GetEnvAsIntOrDefault(dataStoreGeneralMetadataCacheSizeMBVarName, defaultDataStoreGeneralMetadataCacheSizeMB)
}

type GeneralCommandArgs struct {
	*CommandArgs
	SubCommands      []string
	LoggableFlag     []string
	LoggableKV       map[string]string
	RedactedKV       map[string]string
	OutputFileSuffix string
}

// GeneralCommand returns the kopia command
// contains subcommands, loggable flags, loggable key value pairs and
// redacted key value pairs
func GeneralCommand(cmdArgs GeneralCommandArgs) logsafe.Cmd {
	args := commonArgs(cmdArgs.CommandArgs)
	for _, subCmd := range cmdArgs.SubCommands {
		args = args.AppendLoggable(subCmd)
	}
	for _, flag := range cmdArgs.LoggableFlag {
		args = args.AppendLoggable(flag)
	}
	for k, v := range cmdArgs.LoggableKV {
		args = args.AppendLoggableKV(k, v)
	}
	for k, v := range cmdArgs.RedactedKV {
		args = args.AppendRedactedKV(k, v)
	}
	return args
}
