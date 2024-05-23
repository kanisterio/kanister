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

	"github.com/go-openapi/strfmt"
	"github.com/kanisterio/safecli"
	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
)

func TestRepositoryConnectCommand(t *testing.T) { check.TestingT(t) }

// Test Repository Connect command
var _ = check.Suite(test.NewCommandSuite([]test.CommandTest{
	{
		Name: "repository connect with default retention",
		Command: func() (*safecli.Builder, error) {
			args := ConnectArgs{
				Common:         common,
				Cache:          cache,
				Hostname:       "test-hostname",
				Username:       "test-username",
				RepoPathPrefix: "test-path/prefix",
				Location:       locFS,
			}
			return Connect(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-dir=cache/log",
			"--log-level=error",
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
		Name: "repository connect with ReadOnly",
		Command: func() (*safecli.Builder, error) {
			args := ConnectArgs{
				Common:         common,
				Cache:          cache,
				Hostname:       "test-hostname",
				Username:       "test-username",
				RepoPathPrefix: "test-path/prefix",
				Location:       locFS,
				ReadOnly:       true,
			}
			return Connect(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-dir=cache/log",
			"--log-level=error",
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
		},
	},
	{
		Name: "repository connect with PIT and ReadOnly",
		Command: func() (*safecli.Builder, error) {
			pit, _ := strfmt.ParseDateTime("2021-02-03T01:02:03Z")
			args := ConnectArgs{
				Common:         common,
				Cache:          cache,
				Hostname:       "test-hostname",
				Username:       "test-username",
				RepoPathPrefix: "path/prefix",
				Location:       locS3,
				PointInTime:    pit,
				ReadOnly:       true,
			}
			return Connect(args)
		},
		ExpectedCLI: []string{"kopia",
			"--config-file=path/kopia.config",
			"--log-dir=cache/log",
			"--log-level=error",
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
			"s3",
			"--region=test-region",
			"--bucket=test-bucket",
			"--endpoint=test-endpoint",
			"--prefix=test-prefix/path/prefix/",
			"--point-in-time=2021-02-03T01:02:03.000Z",
		},
	},
}))
