// Copyright 2024 The Kanister Authors.
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

package opts

import (
	"github.com/kanisterio/kanister/pkg/kopia/cli/args"
	"github.com/kanisterio/safecli/command"
)

const (
	defaultLogLevel = "error"
)

// LogDirectory creates a new log directory option with a given directory.
func LogDirectory(dir string) command.Applier {
	return command.NewOptionWithArgument("--log-dir", dir)
}

// LogLevel creates a new log level flag with a given level.
// If the level is empty, the default log level is used.
func LogLevel(level string) command.Applier {
	if level == "" {
		level = defaultLogLevel
	}
	return command.NewOptionWithArgument("--log-level", level)
}

// ConfigFilePath creates a new config file path option with a given path.
func ConfigFilePath(path string) command.Applier {
	return command.NewOptionWithArgument("--config-file", path)
}

// RepoPassword creates a new repository password option with a given password.
func RepoPassword(password string) command.Applier {
	return command.NewOptionWithRedactedArgument("--password", password)
}

// Common maps the common arguments to the CLI command options.
func Common(args args.Common) command.Applier {
	return command.NewArguments(
		ConfigFilePath(args.ConfigFilePath),
		LogDirectory(args.LogDirectory),
		LogLevel(args.LogLevel),
		RepoPassword(args.RepoPassword),
	)
}