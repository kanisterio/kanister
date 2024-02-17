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

package s3

import (
	"testing"

	"github.com/kanisterio/safecli/command"
	"github.com/kanisterio/safecli/test"
	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal"
)

func TestNewS3(t *testing.T) { check.TestingT(t) }

func newS3(prefix, repoPath, endpoint string) command.Applier {
	l := internal.Location{
		"prefix":        []byte(prefix),
		"endpoint":      []byte(endpoint),
		"region":        []byte("region"),
		"bucket":        []byte("bucket"),
		"skipSSLVerify": []byte("true"),
	}
	return New(l, repoPath, nil)
}

var _ = check.Suite(&test.ArgumentSuite{Cmd: "cmd", Arguments: []test.ArgumentTest{
	{
		Name:     "NewS3",
		Argument: newS3("prefix", "repoPath", "http://endpoint/path/"),
		ExpectedCLI: []string{"cmd", "s3",
			"--region=region",
			"--bucket=bucket",
			"--endpoint=endpoint/path",
			"--prefix=prefix/repoPath/",
			"--disable-tls",
			"--disable-tls-verification",
		},
	},
	{
		Name:     "NewS3 with empty repoPath and https endpoint",
		Argument: newS3("prefix", "", "https://endpoint/path/"),
		ExpectedCLI: []string{"cmd", "s3",
			"--region=region",
			"--bucket=bucket",
			"--endpoint=endpoint/path",
			"--prefix=prefix/",
			"--disable-tls-verification",
		},
	},
	{
		Name:     "NewS3 with empty repoPath and endpoint",
		Argument: newS3("prefix", "", ""),
		ExpectedCLI: []string{"cmd", "s3",
			"--region=region",
			"--bucket=bucket",
			"--prefix=prefix/",
			"--disable-tls-verification",
		},
	},
	{
		Name:        "NewS3 with empty local prefix and repo prefix should return error",
		Argument:    newS3("", "", ""),
		ExpectedErr: cli.ErrInvalidRepoPath,
	},
}})
