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

	"github.com/kanisterio/safecli"
	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/args"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
)

func TestRepositoryCreateCommand(t *testing.T) { check.TestingT(t) }

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
	{
		Name: "repository create with filestore location and additional args",
		Command: func() (*safecli.Builder, error) {
			arguments := CreateArgs{
				Common:          common,
				Cache:           cache,
				Hostname:        "test-hostname",
				Username:        "test-username",
				RepoPathPrefix:  "test-path/prefix",
				Location:        locFS,
				RetentionMode:   retentionMode,
				RetentionPeriod: retentionPeriod,
			}
			flags := args.RepositoryCreate
			args.RepositoryCreate = args.Args{}
			args.RepositoryCreate.Set("--testflag", "testvalue")
			defer func() { args.RepositoryCreate = flags }()

			return Create(arguments)
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
			"--testflag=testvalue",
		},
	},
}))
