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
	"os"

	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/logsafe"
)

const (
	// LogLevelVarName is the environment variable that controls datastore log level.
	LogLevelVarName = "DATA_STORE_LOG_LEVEL"
	// FileLogLevelVarName is the environment variable that controls datastore file log level.
	FileLogLevelVarName = "DATA_STORE_FILE_LOG_LEVEL"
)

func NonEmptyOrDefault[T comparable](t T, def T) T {
	var empty T
	if t != empty {
		return t
	}
	return def
}

func GetEnvOrDefault(name, def string) string {
	return NonEmptyOrDefault(os.Getenv(name), def)
}

// LogLevel will return either value from env or "error" as default value
func LogLevel() string {
	return GetEnvOrDefault(LogLevelVarName, LogLevelError)
}

// FileLogLevel will return value from env
func FileLogLevel() string {
	return os.Getenv(FileLogLevelVarName)
}

type CommandArgs struct {
	RepoPassword   string
	ConfigFilePath string
	LogDirectory   string
	LogLevel       string
	FileLogLevel   string
}

func bashCommand(args logsafe.Cmd) []string {
	log.Info().Print("Kopia Command", field.M{"Command": args.String()})
	return []string{"bash", "-o", "errexit", "-c", args.PlainText()}
}

func stringSliceCommand(args logsafe.Cmd) []string {
	log.Info().Print("Kopia Command", field.M{"Command": args.String()})
	return args.StringSliceCMD()
}

func commonArgs(cmdArgs *CommandArgs) logsafe.Cmd {
	c := logsafe.NewLoggable(kopiaCommand)

	logLevel := NonEmptyOrDefault(cmdArgs.LogLevel, LogLevel())
	c = c.AppendLoggableKV(logLevelFlag, logLevel)

	fileLogLevel := NonEmptyOrDefault(cmdArgs.FileLogLevel, FileLogLevel())
	if fileLogLevel != "" {
		c = c.AppendLoggableKV(fileLogLevelFlag, fileLogLevel)
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

func addTags(tags []string, args logsafe.Cmd) logsafe.Cmd {
	// kopia required tags in name:value format, but all checks are performed on kopia side
	for _, tag := range tags {
		args = args.AppendLoggable(tagsFlag, tag)
	}
	return args
}

// ExecKopiaArgs returns the basic Argv for executing kopia with the given config file path.
func ExecKopiaArgs(configFilePath string) []string {
	return commonArgs(&CommandArgs{ConfigFilePath: configFilePath}).StringSliceCMD()
}

const (
	cacheDirectoryFlag           = "--cache-directory"
	contentCacheSizeMBFlag       = "--content-cache-size-mb"
	metadataCacheSizeMBFlag      = "--metadata-cache-size-mb"
	contentCacheSizeLimitMBFlag  = "--content-cache-size-limit-mb"
	metadataCacheSizeLimitMBFlag = "--metadata-cache-size-limit-mb"
)
