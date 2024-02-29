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
	"strconv"

	"github.com/kanisterio/safecli/command"

	"github.com/kanisterio/kanister/pkg/kopia/cli/args"
)

const (
	defaultCacheDirectory = "/tmp/kopia-cache"
)

// CacheDirectory creates a new cache directory option with a given directory.
// If the directory is empty, the default cache directory is used.
func CacheDirectory(dir string) command.Applier {
	if dir == "" {
		dir = defaultCacheDirectory
	}
	return command.NewOptionWithArgument("--cache-directory", dir)
}

// ContentCacheSizeLimitMB creates a new content cache size option with a given size.
func ContentCacheSizeLimitMB(size int) command.Applier {
	val := strconv.Itoa(size)
	return command.NewOptionWithArgument("--content-cache-size-limit-mb", val)
}

// MetadataCacheSizeLimitMB creates a new metadata cache size option with a given size.
func MetadataCacheSizeLimitMB(size int) command.Applier {
	val := strconv.Itoa(size)
	return command.NewOptionWithArgument("--metadata-cache-size-limit-mb", val)
}

// ContentCacheSizeMB creates a new content cache size option with a given size.
func ContentCacheSizeMB(size int) command.Applier {
	val := strconv.Itoa(size)
	return command.NewOptionWithArgument("--content-cache-size-mb", val)
}

// MetadataCacheSizeMB creates a new metadata cache size option with a given size.
func MetadataCacheSizeMB(size int) command.Applier {
	val := strconv.Itoa(size)
	return command.NewOptionWithArgument("--metadata-cache-size-mb", val)
}

// Cache maps the Cache arguments to the CLI command options.
func Cache(args args.Cache) command.Applier {
	return command.NewArguments(
		CacheDirectory(args.CacheDirectory),
		ContentCacheSizeLimitMB(args.ContentCacheSizeLimitMB),
		MetadataCacheSizeLimitMB(args.MetadataCacheSizeLimitMB),
	)
}
