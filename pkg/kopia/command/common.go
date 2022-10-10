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

	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/logsafe"
)

type CommandArgs struct {
	RepoPassword   string
	ConfigFilePath string
	LogDirectory   string
}

func bashCommand(args logsafe.Cmd) []string {
	log.Debug().Print("Kopia Command", field.M{"Command": args.String()})
	return []string{"bash", "-o", "errexit", "-c", args.PlainText()}
}

func stringSliceCommand(args logsafe.Cmd) []string {
	log.Debug().Print("Kopia Command", field.M{"Command": args.String()})
	return args.StringSliceCMD()
}

func commonArgs(cmdArgs *CommandArgs, requireInfoLevel bool) logsafe.Cmd {
	c := logsafe.NewLoggable(kopiaCommand)
	if requireInfoLevel {
		c = c.AppendLoggable(logLevelInfoFlag)
	} else {
		c = c.AppendLoggable(logLevelErrorFlag)
	}
	if cmdArgs.ConfigFilePath != "" {
		c = c.AppendLoggableKV(configFileFlag, cmdArgs.ConfigFilePath)
	}
	if cmdArgs.LogDirectory != "" {
		c = c.AppendLoggableKV(logDirectoryFlag, cmdArgs.LogDirectory)
	}
	if cmdArgs.RepoPassword != "" {
		c = c.AppendRedactedKV(passwordFlag, cmdArgs.RepoPassword)
	}
	return c
}

// ExecKopiaArgs returns the basic Argv for executing kopia with the given config file path.
func ExecKopiaArgs(configFilePath string) []string {
	return commonArgs(&CommandArgs{ConfigFilePath: configFilePath}, false).StringSliceCMD()
}

const (
	cacheDirectoryFlag      = "--cache-directory"
	contentCacheSizeMBFlag  = "--content-cache-size-mb"
	metadataCacheSizeMBFlag = "--metadata-cache-size-mb"
)

func kopiaCacheArgs(args logsafe.Cmd, cacheDirectory string, contentCacheMB, metadataCacheMB int) logsafe.Cmd {
	args = args.AppendLoggableKV(cacheDirectoryFlag, cacheDirectory)
	args = args.AppendLoggableKV(contentCacheSizeMBFlag, strconv.Itoa(contentCacheMB))
	args = args.AppendLoggableKV(metadataCacheSizeMBFlag, strconv.Itoa(metadataCacheMB))
	return args
}
