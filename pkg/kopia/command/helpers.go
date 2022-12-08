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

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"

	"github.com/kanisterio/kanister/pkg/kopia"
	"github.com/kanisterio/kanister/pkg/kopia/command/storage"
	"github.com/kanisterio/kanister/pkg/logsafe"
	"github.com/kanisterio/kanister/pkg/utils"
)

type policyChanges map[string]string

// GetCacheSizeSettingsForSnapshot returns the feature setting cache size values to be used
// for initializing repositories that will be performing general command workloads that benefit from
// cacheing metadata only.
func GetCacheSizeSettingsForSnapshot() (contentCacheMB, metadataCacheMB int) {
	return utils.GetEnvAsIntOrDefault(kopia.DataStoreGeneralContentCacheSizeMBVarName, kopia.DefaultDataStoreGeneralContentCacheSizeMB),
		utils.GetEnvAsIntOrDefault(kopia.DataStoreGeneralMetadataCacheSizeMBVarName, kopia.DefaultDataStoreGeneralMetadataCacheSizeMB)
}

// GetCacheSizeSettingsForRestore returns the feature setting cache size values to be used
// for initializing repositories that will be performing restore workloads
func GetCacheSizeSettingsForRestore() (contentCacheMB, metadataCacheMB int) {
	return utils.GetEnvAsIntOrDefault(kopia.DataStoreRestoreContentCacheSizeMBVarName, kopia.DefaultDataStoreRestoreContentCacheSizeMB),
		utils.GetEnvAsIntOrDefault(kopia.DataStoreRestoreMetadataCacheSizeMBVarName, kopia.DefaultDataStoreRestoreMetadataCacheSizeMB)
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
	args := commonArgs(cmdArgs.CommandArgs, false)
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

// GenerateEnvSpecFromRepoPasswordSecret returns envvar generated
// for repository password secret
func GenerateEnvSpecFromRepoPasswordSecret(s *v1.Secret) (*v1.EnvVar, error) {
	if s == nil {
		return nil, errors.New("Secret cannot be nil")
	}
	if s.Data == nil {
		return nil, errors.New("Secret data cannot be nil")
	}
	if _, ok := s.Data[RepoPassordKey]; !ok {
		return nil, errors.New(fmt.Sprint("Repository password key not set: ", RepoPassordKey))
	}
	envVar := storage.GetEnvVarWithSecretRef(RepoPassordKey, s.Name, KopiaRepoPasswordEnv)
	return &envVar, nil
}
