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

package gcs

import (
	"testing"

	"gopkg.in/check.v1"

	"github.com/kanisterio/safecli"

	rs "github.com/kanisterio/kanister/pkg/secrets/repositoryserver"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/storage/model"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
)

func TestStorageGCS(t *testing.T) { check.TestingT(t) }

var _ = check.Suite(test.NewCommandSuite([]test.CommandTest{
	{
		Name: "Empty GCS storage flag should generate subcommand with default flags",
		CLI: func() (safecli.CommandBuilder, error) {
			return New(model.StorageFlag{})
		},
		ExpectedCLI: []string{
			"gcs",
			"--credentials-file=/tmp/creds.txt",
		},
	},
	{
		Name: "GCS with values should generate subcommand with specific flags",
		CLI: func() (safecli.CommandBuilder, error) {
			return New(model.StorageFlag{
				RepoPathPrefix: "repo/path/prefix",
				Location: model.Location{
					rs.PrefixKey: []byte("prefix"),
					rs.BucketKey: []byte("bucket"),
				},
			})
		},
		ExpectedCLI: []string{
			"gcs",
			"--bucket=bucket",
			"--credentials-file=/tmp/creds.txt",
			"--prefix=prefix/repo/path/prefix/",
		},
	},
}))
