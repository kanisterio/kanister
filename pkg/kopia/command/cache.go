// Copyright 2023 The Kanister Authors.
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

	"github.com/kanisterio/kanister/pkg/logsafe"
)

// CacheArgs has fields that can be used to set
// cache settings for different kopia repository operations
type CacheArgs struct {
	ContentCacheLimitMB  int
	MetadataCacheLimitMB int
}

func (c CacheArgs) kopiaCacheArgs(args logsafe.Cmd, cacheDirectory string) logsafe.Cmd {
	args = args.AppendLoggableKV(cacheDirectoryFlag, cacheDirectory)

	// The hard limit flags for cache would be set using the env variables that
	// are passed through Kanister helm settings.
	// Soft limit flags would be set to the default values by kopia
	// automatically in ``connectOptions.setup()`` function.
	// Refer to https://github.com/kopia/kopia/blob/master/cli/command_repository_connect.go#L71
	args = args.AppendLoggableKV(contentCacheSizeLimitMBFlag, strconv.Itoa(c.ContentCacheLimitMB))
	args = args.AppendLoggableKV(metadataCacheSizeLimitMBFlag, strconv.Itoa(c.MetadataCacheLimitMB))
	return args
}
