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

package azure

import (
	"testing"

	"github.com/kanisterio/safecli/command"
	"github.com/kanisterio/safecli/test"
	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal"
)

func TestNewAzure(t *testing.T) { check.TestingT(t) }

func newAzure(prefix, repoPath, bucket string) command.Applier {
	l := internal.Location{
		"prefix": []byte(prefix),
		"bucket": []byte(bucket),
	}
	return New(l, repoPath, nil)
}

var _ = check.Suite(&test.ArgumentSuite{Cmd: "cmd", Arguments: []test.ArgumentTest{
	{
		Name:        "NewAzure",
		Argument:    newAzure("prefix", "repoPath", "bucket"),
		ExpectedCLI: []string{"cmd", "azure", "--container=bucket", "--prefix=prefix/repoPath/"},
	},
	{
		Name:        "NewAzure with empty repoPath",
		Argument:    newAzure("prefix", "", "bucket"),
		ExpectedCLI: []string{"cmd", "azure", "--container=bucket", "--prefix=prefix/"},
	},
	{
		Name:        "NewAzure with empty local prefix and repo prefix should return error",
		Argument:    newAzure("", "", "bucket"),
		ExpectedCLI: []string{"cmd", "azure", "--container=bucket", "--prefix="},
	},
	{
		Name:        "NewAzure with empty bucket should return ErrInvalidContainerName",
		Argument:    newAzure("", "", ""),
		ExpectedErr: cli.ErrInvalidContainerName,
	},
}})
