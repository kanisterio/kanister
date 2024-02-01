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

package common

import (
	"strconv"

	"github.com/kanisterio/safecli"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag"
)

// Flags without conditions which applied to different kopia commands.
var (
	All           = flag.NewBoolFlag("--all", true)
	Delta         = flag.NewBoolFlag("--delta", true)
	ShowIdentical = flag.NewBoolFlag("--show-identical", true)
	NoGRPC        = flag.NewBoolFlag("--no-grpc", true)
)

// predefined flags
var (
	CheckForUpdates   = checkForUpdates{CheckForUpdates: true}
	NoCheckForUpdates = checkForUpdates{CheckForUpdates: false}
)

// flag defaults
var (
	defaultLogLevel        = "error"
	defaultCacheDirectory  = "/tmp/kopia-cache"
	defaultConfigDirectory = "/tmp/kopia-repository"
)

// LogDirectory creates a new log directory flag with a given directory.
func LogDirectory(dir string) flag.Applier {
	return flag.NewStringFlag("--log-dir", dir)
}

// LogLevel creates a new log level flag with a given level.
// If the level is empty, the default log level is used.
func LogLevel(level string) flag.Applier {
	if level == "" {
		level = defaultLogLevel
	}
	return flag.NewStringFlag("--log-level", level)
}

// CacheDirectory creates a new cache directory flag with a given directory.
// If the directory is empty, the default cache directory is used.
func CacheDirectory(dir string) flag.Applier {
	if dir == "" {
		dir = defaultCacheDirectory
	}
	return flag.NewStringFlag("--cache-directory", dir)
}

// ConfigFilePath creates a new config file path flag with a given path.
func ConfigFilePath(path string) flag.Applier {
	return flag.NewStringFlag("--config-file", path)
}

// ConfigDirectory creates a new config directory flag with a given directory.
// If the directory is empty, the default config directory is used.
func ConfigDirectory(dir string) flag.Applier {
	if dir == "" {
		dir = defaultConfigDirectory
	}
	return flag.NewStringFlag("--config-directory", dir)
}

// RepoPassword creates a new repository password flag with a given password.
func RepoPassword(password string) flag.Applier {
	return flag.NewRedactedStringFlag("--password", password)
}

// checkForUpdates is the flag for checking for updates.
type checkForUpdates struct {
	CheckForUpdates bool
}

func (f checkForUpdates) Flag() string {
	if f.CheckForUpdates {
		return "--check-for-updates"
	}
	return "--no-check-for-updates"
}

func (f checkForUpdates) Apply(cli safecli.CommandAppender) error {
	cli.AppendLoggable(f.Flag())
	return nil
}

// ReadOnly creates a new read only flag.
func ReadOnly(readOnly bool) flag.Applier {
	return flag.NewBoolFlag("--readonly", readOnly)
}

// ContentCacheSizeLimitMB creates a new content cache size flag with a given size.
func ContentCacheSizeLimitMB(size int) flag.Applier {
	val := strconv.Itoa(size)
	return flag.NewStringFlag("--content-cache-size-limit-mb", val)
}

// ContentCacheSizeMB creates a new content cache size flag with a given size.
func ContentCacheSizeMB(size int) flag.Applier {
	val := strconv.Itoa(size)
	return flag.NewStringFlag("--content-cache-size-mb", val)
}

// MetadataCacheSizeLimitMB creates a new metadata cache size flag with a given size.
func MetadataCacheSizeLimitMB(size int) flag.Applier {
	val := strconv.Itoa(size)
	return flag.NewStringFlag("--metadata-cache-size-limit-mb", val)
}

// MetadataCacheSizeMB creates a new metadata cache size flag with a given size.
func MetadataCacheSizeMB(size int) flag.Applier {
	val := strconv.Itoa(size)
	return flag.NewStringFlag("--metadata-cache-size-mb", val)
}

// common are the global flags for Kopia.
type common struct {
	cli.CommonArgs
}

// Apply applies the global flags to the command.
func (f common) Apply(cmd safecli.CommandAppender) error {
	return flag.Apply(cmd,
		LogLevel(f.LogLevel),
		ConfigFilePath(f.ConfigFilePath),
		LogDirectory(f.LogDirectory),
		RepoPassword(f.RepoPassword),
	)
}

// Common creates a new common flag.
// If no arguments are provided, the default common flags are used.
// If one argument is provided, the common flags are used.
// If more than one argument is provided, ErrInvalidCommonArgs is returned.
func Common(args ...cli.CommonArgs) flag.Applier {
	if len(args) == 0 {
		return common{cli.CommonArgs{}}
	} else if len(args) == 1 {
		return common{args[0]}
	} else {
		return flag.ErrorFlag(cli.ErrInvalidCommonArgs)
	}
}

// cache defines cache flags and implements Applier interface for the cache flags.
type cache struct {
	cli.CacheArgs
}

// Apply applies the cache flags to the command.
func (f cache) Apply(cmd safecli.CommandAppender) error {
	return flag.Apply(cmd,
		CacheDirectory(f.CacheDirectory),
		ContentCacheSizeLimitMB(f.ContentCacheSizeLimitMB),
		MetadataCacheSizeLimitMB(f.MetadataCacheSizeLimitMB),
		// ContentCacheSizeMB(f.ContentCacheSizeMB),
		// MetadataCacheSizeMB(f.MetadataCacheSizeMB),
	)
}

// Cache creates a new cache flag.
// If no arguments are provided, the default cache flags are used.
// If one argument is provided, the cache flags are used.
// If more than one argument is provided, ErrInvalidCacheArgs is returned.
func Cache(args ...cli.CacheArgs) flag.Applier {
	if len(args) == 0 {
		return cache{cli.CacheArgs{}}
	} else if len(args) == 1 {
		return cache{args[0]}
	} else {
		return flag.ErrorFlag(cli.ErrInvalidCacheArgs)
	}
}

// JSONOutput creates a new JSON output flag.
func JSONOutput(enable bool) flag.Applier {
	return flag.NewBoolFlag("--json", enable)
}

// JSON flag enables JSON output for different kopia commands.
var JSON = JSONOutput(true)

// Delete creates a new delete flag.
func Delete(enable bool) flag.Applier {
	return flag.NewBoolFlag("--delete", enable)
}

// ID create the Kopia ID argument for different commands.
func ID(id string) flag.Applier {
	if id == "" {
		return flag.ErrorFlag(cli.ErrInvalidID)
	}
	return flag.NewStringArgument(id)
}
