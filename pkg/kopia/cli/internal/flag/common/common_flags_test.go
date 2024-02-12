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
	"fmt"
	"testing"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
	"gopkg.in/check.v1"
)

func TestCommonFlags(t *testing.T) { check.TestingT(t) }

var _ = check.Suite(test.NewFlagSuite([]test.FlagTest{
	{
		Name: "Empty LogDirectory should generate a flag with default value",
		Flag: LogDirectory(""),
	},
	{
		Name:        "LogDirectory with value should generate a flag with the given directory",
		Flag:        LogDirectory("/path/to/logs"),
		ExpectedCLI: []string{"--log-dir=/path/to/logs"},
	},
	{
		Name:        "Empty LogLevel should generate a flag with default value",
		Flag:        LogLevel(""),
		ExpectedCLI: []string{fmt.Sprintf("--log-level=%s", defaultLogLevel)},
	},
	{
		Name:        "LogLevel with value should generate a flag with the given level",
		Flag:        LogLevel("info"),
		ExpectedCLI: []string{"--log-level=info"},
	},
	{
		Name:        "Empty CacheDirectory should generate a flag with default value",
		Flag:        CacheDirectory(""),
		ExpectedCLI: []string{fmt.Sprintf("--cache-directory=%s", defaultCacheDirectory)},
	},
	{
		Name:        "CacheDirectory with value should generate a flag with the given directory",
		Flag:        CacheDirectory("/home/user/.cache/kopia"),
		ExpectedCLI: []string{"--cache-directory=/home/user/.cache/kopia"},
	},
	{
		Name: "Empty ConfigFilePath should not generate a flag",
		Flag: ConfigFilePath(""),
	},
	{
		Name:        "ConfigFilePath with value should generate a flag with the given config file path",
		Flag:        ConfigFilePath("/var/kopia/config"),
		ExpectedCLI: []string{"--config-file=/var/kopia/config"},
	},
	{
		Name: "Empty RepoPassword should not generate a flag",
		Flag: RepoPassword(""),
	},
	{
		Name:        "RepoPassword with value should generate a flag with the given value and redact it for logs",
		Flag:        RepoPassword("pass12345"),
		ExpectedCLI: []string{"--password=pass12345"},
	},
	{
		Name:        "CheckForUpdates should always generate a flag",
		Flag:        CheckForUpdates,
		ExpectedCLI: []string{"--check-for-updates"},
	},
	{
		Name:        "NoCheckForUpdates should always generate a flag",
		Flag:        NoCheckForUpdates,
		ExpectedCLI: []string{"--no-check-for-updates"},
	},
	{
		Name: "ReadOnly(false)should not generate a flag",
		Flag: ReadOnly(false),
	},
	{
		Name:        "ReadOnly(true) should generate a flag",
		Flag:        ReadOnly(true),
		ExpectedCLI: []string{"--readonly"},
	},
	{
		Name:        "NoGRPC should always generate '--no-grpc' flag",
		Flag:        NoGRPC,
		ExpectedCLI: []string{"--no-grpc"},
	},
	{
		Name:        "JSON should always generate a flag",
		Flag:        JSON,
		ExpectedCLI: []string{"--json"},
	},
	{
		Name:        "ContentCacheSizeLimitMB with value should generate a flag with the given value",
		Flag:        ContentCacheSizeLimitMB(1024),
		ExpectedCLI: []string{"--content-cache-size-limit-mb=1024"},
	},
	{
		Name:        "ContentCacheSizeMB with value should generate a flag with the given value",
		Flag:        ContentCacheSizeMB(1024),
		ExpectedCLI: []string{"--content-cache-size-mb=1024"},
	},
	{
		Name:        "MetadataCacheSizeLimitMB with value should generate a flag with the given value",
		Flag:        MetadataCacheSizeLimitMB(1024),
		ExpectedCLI: []string{"--metadata-cache-size-limit-mb=1024"},
	},
	{
		Name:        "MetadataCacheSizeMB with value should generate a flag with the given value",
		Flag:        MetadataCacheSizeMB(1024),
		ExpectedCLI: []string{"--metadata-cache-size-mb=1024"},
	},
	{
		Name:        "Empty Common should generate a flag with default value(s)",
		Flag:        Common(cli.CommonArgs{}),
		ExpectedCLI: []string{"--log-level=error"},
	},
	{
		Name: "Common with values should generate multiple flags with the given values and redact password for logs",
		Flag: Common(cli.CommonArgs{
			ConfigFilePath: "/var/kopia/config",
			LogDirectory:   "/var/log/kopia",
			LogLevel:       "info",
			RepoPassword:   "pass12345",
		}),
		ExpectedCLI: []string{
			"--config-file=/var/kopia/config",
			"--log-level=info",
			"--log-dir=/var/log/kopia",
			"--password=pass12345",
		},
	},
	{
		Name: "Empty FlagCacheArgs should generate multiple flags with default values",
		Flag: Cache(cli.CacheArgs{}),
		ExpectedCLI: []string{
			"--cache-directory=/tmp/kopia-cache",
			"--content-cache-size-limit-mb=0",
			"--metadata-cache-size-limit-mb=0",
		},
	},
	{
		Name: "Cache with CacheArgs should generate multiple cache related flags",
		Flag: Cache(cli.CacheArgs{
			CacheDirectory:           "/home/user/.cache/kopia",
			ContentCacheSizeLimitMB:  1024,
			MetadataCacheSizeLimitMB: 2048,
		}),
		ExpectedCLI: []string{
			"--cache-directory=/home/user/.cache/kopia",
			"--content-cache-size-limit-mb=1024",
			"--metadata-cache-size-limit-mb=2048",
		},
	},
	{
		Name: "Delete(false) should not generate a flag",
		Flag: Delete(false),
	},
	{
		Name:        "Delete(true) should generate a flag",
		Flag:        Delete(true),
		ExpectedCLI: []string{"--delete"},
	},
	{
		Name:        "Empty ID should generate an ErrInvalidID error",
		Flag:        ID(""),
		ExpectedErr: cli.ErrInvalidID,
	},
	{
		Name:        "ID with value should generate an argument with the given value",
		Flag:        ID("id12345"),
		ExpectedCLI: []string{"id12345"},
	},
}))
