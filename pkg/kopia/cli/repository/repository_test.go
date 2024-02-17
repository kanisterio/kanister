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

	"github.com/kanisterio/safecli"
	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/args"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
	rs "github.com/kanisterio/kanister/pkg/secrets/repositoryserver"
)

func TestRepositoryCommands(t *testing.T) { check.TestingT(t) }

var (
	common = args.Common{
		RepoPassword:   "encr-key",
		ConfigFilePath: "path/kopia.config",
		LogDirectory:   "cache/log",
	}

	cache = args.Cache{
		CacheDirectory:           "/tmp/cache.dir",
		ContentCacheSizeLimitMB:  0,
		MetadataCacheSizeLimitMB: 0,
	}

	retentionMode   = "Locked"
	retentionPeriod = 15 * time.Minute

	locFS = internal.Location{
		rs.TypeKey:   []byte("filestore"),
		rs.PrefixKey: []byte("test-prefix"),
	}

	locAzure = internal.Location{
		rs.TypeKey:   []byte("azure"),
		rs.BucketKey: []byte("test-bucket"),
		rs.PrefixKey: []byte("test-prefix"),
	}

	locGCS = internal.Location{
		rs.TypeKey:   []byte("gcs"),
		rs.BucketKey: []byte("test-bucket"),
		rs.PrefixKey: []byte("test-prefix"),
	}

	locS3 = internal.Location{
		rs.TypeKey:          []byte("s3"),
		rs.EndpointKey:      []byte("test-endpoint"),
		rs.RegionKey:        []byte("test-region"),
		rs.BucketKey:        []byte("test-bucket"),
		rs.PrefixKey:        []byte("test-prefix"),
		rs.SkipSSLVerifyKey: []byte("false"),
	}

	locS3Compliant = internal.Location{
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
		Name: "repository create with no storage",
		Command: func() (*safecli.Builder, error) {
			args := CreateArgs{
				Common:         common,
				Cache:          cache,
				Hostname:       "test-hostname",
				Username:       "test-username",
				RepoPathPrefix: "test-path/prefix",
			}
			return Create(args)
		},
		ExpectedErr: cli.ErrUnsupportedStorage,
	},
	{
		Name: "repository create with filestore location",
		Command: func() (*safecli.Builder, error) {
			args := CreateArgs{
				Common:          common,
				Cache:           cache,
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
			"--config-file=path/kopia.config",
			"--log-dir=cache/log",
			"--log-level=error",
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
	{
		Name: "repository create with azure location",
		Command: func() (*safecli.Builder, error) {
			args := CreateArgs{
				Common:          common,
				Cache:           cache,
				Hostname:        "test-hostname",
				Username:        "test-username",
				RepoPathPrefix:  "test-path/prefix",
				Location:        locAzure,
				RetentionMode:   retentionMode,
				RetentionPeriod: retentionPeriod,
			}
			return Create(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-dir=cache/log",
			"--log-level=error",
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
			"azure",
			"--container=test-bucket",
			"--prefix=test-prefix/test-path/prefix/",
		},
	},
	{
		Name: "repository create with gcs location",
		Command: func() (*safecli.Builder, error) {
			args := CreateArgs{
				Common:          common,
				Cache:           cache,
				Hostname:        "test-hostname",
				Username:        "test-username",
				RepoPathPrefix:  "test-path/prefix",
				Location:        locGCS,
				RetentionMode:   retentionMode,
				RetentionPeriod: retentionPeriod,
			}
			return Create(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-dir=cache/log",
			"--log-level=error",
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
			"gcs",
			"--bucket=test-bucket",
			"--credentials-file=/tmp/creds.txt",
			"--prefix=test-prefix/test-path/prefix/",
		},
	},
	{
		Name: "repository create with s3 location",
		Command: func() (*safecli.Builder, error) {
			args := CreateArgs{
				Common:          common,
				Cache:           cache,
				Hostname:        "test-hostname",
				Username:        "test-username",
				RepoPathPrefix:  "test-path/prefix",
				Location:        locS3,
				RetentionMode:   retentionMode,
				RetentionPeriod: retentionPeriod,
			}
			return Create(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-dir=cache/log",
			"--log-level=error",
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
			"s3",
			"--region=test-region",
			"--bucket=test-bucket",
			"--endpoint=test-endpoint",
			"--prefix=test-prefix/test-path/prefix/",
		},
	},
	{
		Name: "repository create with s3 compliant location",
		Command: func() (*safecli.Builder, error) {
			args := CreateArgs{
				Common:          common,
				Cache:           cache,
				Hostname:        "test-hostname",
				Username:        "test-username",
				RepoPathPrefix:  "test-path/prefix",
				Location:        locS3Compliant,
				RetentionMode:   retentionMode,
				RetentionPeriod: retentionPeriod,
			}
			return Create(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-dir=cache/log",
			"--log-level=error",
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
			"s3",
			"--region=test-region",
			"--bucket=test-bucket",
			"--endpoint=test-endpoint",
			"--prefix=test-prefix/test-path/prefix/",
		},
	},
}))
