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

package repository

import (
	"testing"
	"time"

	"gopkg.in/check.v1"

	"github.com/go-openapi/strfmt"

	"github.com/kanisterio/safecli"

	rs "github.com/kanisterio/kanister/pkg/secrets/repositoryserver"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/storage/model"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
)

func TestRepositoryCommands(t *testing.T) { check.TestingT(t) }

var (
	cacheArgs = cli.CacheArgs{
		CacheDirectory:           "/tmp/cache.dir",
		ContentCacheSizeLimitMB:  0,
		MetadataCacheSizeLimitMB: 0,
	}

	retentionMode   = "Locked"
	retentionPeriod = 15 * time.Minute

	locFS = model.Location{
		rs.TypeKey:   []byte("filestore"),
		rs.PrefixKey: []byte("test-prefix"),
	}

	locAzure = model.Location{
		rs.TypeKey:   []byte("azure"),
		rs.BucketKey: []byte("test-bucket"),
		rs.PrefixKey: []byte("test-prefix"),
	}

	locGCS = model.Location{
		rs.TypeKey:   []byte("gcs"),
		rs.BucketKey: []byte("test-bucket"),
		rs.PrefixKey: []byte("test-prefix"),
	}

	locS3 = model.Location{
		rs.TypeKey:          []byte("s3"),
		rs.EndpointKey:      []byte("test-endpoint"),
		rs.RegionKey:        []byte("test-region"),
		rs.BucketKey:        []byte("test-bucket"),
		rs.PrefixKey:        []byte("test-prefix"),
		rs.SkipSSLVerifyKey: []byte("false"),
	}

	locs3Compliant = model.Location{
		rs.TypeKey:          []byte("s3Compliant"),
		rs.EndpointKey:      []byte("test-endpoint"),
		rs.RegionKey:        []byte("test-region"),
		rs.BucketKey:        []byte("test-bucket"),
		rs.PrefixKey:        []byte("test-prefix"),
		rs.SkipSSLVerifyKey: []byte("false"),
	}
)

// Test Repository Create command
var _ = check.Suite(test.NewCommandSuite([]test.CommandTest{
	{
		Name: "RepositoryCreate with default retention",
		CLI: func() (safecli.CommandBuilder, error) {
			args := CreateArgs{
				CommonArgs:     test.CommonArgs,
				CacheArgs:      cacheArgs,
				Hostname:       "test-hostname",
				Username:       "test-username",
				RepoPathPrefix: "test-path/prefix",
			}
			return Create(args)
		},
		ExpectedErr: cli.ErrUnsupportedStorage,
	},
	{
		Name: "RepositoryCreate with filestore location",
		CLI: func() (safecli.CommandBuilder, error) {
			args := CreateArgs{
				CommonArgs:      test.CommonArgs,
				CacheArgs:       cacheArgs,
				Hostname:        "test-hostname",
				Username:        "test-username",
				RepoPathPrefix:  "test-path/prefix",
				Location:        locFS,
				RetentionMode:   retentionMode,
				RetentionPeriod: retentionPeriod,
			}
			return Create(args)
		},
		ExpectedCLI: []string{"kopia",
			"--log-level=error",
			"--config-file=path/kopia.config",
			"--log-dir=cache/log",
			"--password=encr-key",
			"repository",
			"create",
			"--no-check-for-updates",
			"--cache-directory=/tmp/cache.dir",
			"--content-cache-size-limit-mb=0",
			"--metadata-cache-size-limit-mb=0",
			"--override-hostname=test-hostname",
			"--override-username=test-username",
			"--retention-mode=Locked",
			"--retention-period=15m0s",
			"filesystem",
			"--path=/mnt/data/test-prefix/test-path/prefix/",
		},
	},
}))

// Test Repository Connect command
var _ = check.Suite(test.NewCommandSuite([]test.CommandTest{
	{
		Name: "RepositoryConnect with default retention",
		CLI: func() (safecli.CommandBuilder, error) {
			args := ConnectArgs{
				CommonArgs:     test.CommonArgs,
				CacheArgs:      cacheArgs,
				Hostname:       "test-hostname",
				Username:       "test-username",
				RepoPathPrefix: "test-path/prefix",
				Location:       locFS,
			}
			return Connect(args)
		},
		ExpectedCLI: []string{"kopia",
			"--log-level=error",
			"--config-file=path/kopia.config",
			"--log-dir=cache/log",
			"--password=encr-key",
			"repository",
			"connect",
			"--no-check-for-updates",
			"--cache-directory=/tmp/cache.dir",
			"--content-cache-size-limit-mb=0",
			"--metadata-cache-size-limit-mb=0",
			"--override-hostname=test-hostname",
			"--override-username=test-username",
			"filesystem",
			"--path=/mnt/data/test-prefix/test-path/prefix/",
		},
	},
	{
		Name: "RepositoryConnect with PIT and ReadOnly",
		CLI: func() (safecli.CommandBuilder, error) {
			pit, _ := strfmt.ParseDateTime("2021-02-03T01:02:03Z")
			args := ConnectArgs{
				CommonArgs:     test.CommonArgs,
				CacheArgs:      cacheArgs,
				Hostname:       "test-hostname",
				Username:       "test-username",
				RepoPathPrefix: "test-path/prefix",
				Location:       locFS,
				PointInTime:    pit,
				ReadOnly:       true,
			}
			return Connect(args)
		},
		ExpectedCLI: []string{"kopia",
			"--log-level=error",
			"--config-file=path/kopia.config",
			"--log-dir=cache/log",
			"--password=encr-key",
			"repository",
			"connect",
			"--no-check-for-updates",
			"--readonly",
			"--cache-directory=/tmp/cache.dir",
			"--content-cache-size-limit-mb=0",
			"--metadata-cache-size-limit-mb=0",
			"--override-hostname=test-hostname",
			"--override-username=test-username",
			"filesystem",
			"--path=/mnt/data/test-prefix/test-path/prefix/",
			"--point-in-time=2021-02-03T01:02:03.000Z",
		},
	},
}))

// Test Repository Connect Server command
var _ = check.Suite(test.NewCommandSuite([]test.CommandTest{
	{
		Name: "RepositoryConnectServer with default retention",
		CLI: func() (safecli.CommandBuilder, error) {
			args := ConnectServerArgs{
				CommonArgs:  test.CommonArgs,
				CacheArgs:   cacheArgs,
				Hostname:    "test-hostname",
				Username:    "test-username",
				ServerURL:   "http://test-server",
				Fingerprint: "test-fingerprint",
				ReadOnly:    true,
			}
			return ConnectServer(args)
		},
		ExpectedCLI: []string{"kopia",
			"--log-level=error",
			"--config-file=path/kopia.config",
			"--log-dir=cache/log",
			"--password=encr-key",
			"repository",
			"connect",
			"server",
			"--no-check-for-updates",
			"--no-grpc",
			"--readonly",
			"--cache-directory=/tmp/cache.dir",
			"--content-cache-size-limit-mb=0",
			"--metadata-cache-size-limit-mb=0",
			"--override-hostname=test-hostname",
			"--override-username=test-username",
			"--url=http://test-server",
			"--server-cert-fingerprint=test-fingerprint",
		},
	},
}))

// Test Repository Set Parameters command
var _ = check.Suite(test.NewCommandSuite([]test.CommandTest{
	{
		Name: "RepositorySetParameters with default retention",
		CLI: func() (safecli.CommandBuilder, error) {
			args := SetParametersArgs{
				CommonArgs: test.CommonArgs,
			}
			return SetParameters(args)
		},
		ExpectedCLI: []string{"kopia",
			"--log-level=error",
			"--config-file=path/kopia.config",
			"--log-dir=cache/log",
			"--password=encr-key",
			"repository",
			"set-parameters",
		},
	},
	{
		Name: "RepositorySetParameters with custom retention args",
		CLI: func() (safecli.CommandBuilder, error) {
			args := SetParametersArgs{
				CommonArgs:      test.CommonArgs,
				RetentionMode:   retentionMode,
				RetentionPeriod: retentionPeriod,
			}
			return SetParameters(args)
		},
		ExpectedCLI: []string{"kopia",
			"--log-level=error",
			"--config-file=path/kopia.config",
			"--log-dir=cache/log",
			"--password=encr-key",
			"repository",
			"set-parameters",
			"--retention-mode=Locked",
			"--retention-period=15m0s",
		},
	},
	{
		Name: "RepositorySetParameters with custom retention mode only",
		CLI: func() (safecli.CommandBuilder, error) {
			args := SetParametersArgs{
				CommonArgs:    test.CommonArgs,
				RetentionMode: retentionMode,
			}
			return SetParameters(args)
		},
		ExpectedCLI: []string{"kopia",
			"--log-level=error",
			"--config-file=path/kopia.config",
			"--log-dir=cache/log",
			"--password=encr-key",
			"repository",
			"set-parameters",
			"--retention-mode=Locked",
			"--retention-period=0s",
		},
	},
}))

// Test Repository Status command
var _ = check.Suite(test.NewCommandSuite([]test.CommandTest{
	{
		Name: "RepositoryStatus with default args",
		CLI: func() (safecli.CommandBuilder, error) {
			args := StatusArgs{
				CommonArgs: test.CommonArgs,
			}
			return Status(args)
		},
		ExpectedCLI: []string{"kopia",
			"--log-level=error",
			"--config-file=path/kopia.config",
			"--log-dir=cache/log",
			"--password=encr-key",
			"repository",
			"status",
		},
	},
	{
		Name: "RepositoryStatus with JSON output",
		CLI: func() (safecli.CommandBuilder, error) {
			args := StatusArgs{
				CommonArgs: test.CommonArgs,
				JSONOutput: true,
			}
			return Status(args)
		},
		ExpectedCLI: []string{"kopia",
			"--log-level=error",
			"--config-file=path/kopia.config",
			"--log-dir=cache/log",
			"--password=encr-key",
			"repository",
			"status",
			"--json",
		},
	},
}))
