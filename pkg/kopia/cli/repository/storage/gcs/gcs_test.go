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

	"github.com/kanisterio/safecli/command"
	"github.com/kanisterio/safecli/test"
	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal"
)

func TestNewGCS(t *testing.T) { check.TestingT(t) }

func newGCS(prefix, repoPath, bucket string) command.Applier {
	l := internal.Location{
		"prefix": []byte(prefix),
		"bucket": []byte(bucket),
	}
	return New(l, repoPath, nil)
}

var _ = check.Suite(&test.ArgumentSuite{Cmd: "cmd", Arguments: []test.ArgumentTest{
	{
		Name:        "NewGCS",
		Argument:    newGCS("prefix", "repoPath", "bucket"),
		ExpectedCLI: []string{"cmd", "gcs", "--bucket=bucket", "--credentials-file=/tmp/creds.txt", "--prefix=prefix/repoPath/"},
	},
	{
		Name:        "NewGCS with empty repoPath",
		Argument:    newGCS("prefix", "", "bucket"),
		ExpectedCLI: []string{"cmd", "gcs", "--bucket=bucket", "--credentials-file=/tmp/creds.txt", "--prefix=prefix/"},
	},
	{
		Name:        "NewGCS with empty local prefix and repo prefix should return error",
		Argument:    newGCS("", "", "bucket"),
		ExpectedCLI: []string{"cmd", "gcs", "--bucket=bucket", "--credentials-file=/tmp/creds.txt", "--prefix="},
	},
	{
		Name:        "NewGCS with empty bucket should return ErrInvalidBucketName",
		Argument:    newGCS("", "", ""),
		ExpectedErr: cli.ErrInvalidBucketName,
	},
}})
