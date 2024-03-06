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
)

func TestS3Options(t *testing.T) { check.TestingT(t) }

var _ = check.Suite(&test.ArgumentSuite{Cmd: "cmd", Arguments: []test.ArgumentTest{
	{
		Name:        "optRegion",
		Argument:    command.NewArguments(optRegion("region"), optRegion("")),
		ExpectedCLI: []string{"cmd", "--region=region"},
	},
	{
		Name:        "optBucket with bucketname should return option",
		Argument:    optBucket("bucketname"),
		ExpectedCLI: []string{"cmd", "--bucket=bucketname"},
	},
	{
		Name:        "optBucket with empty bucketname should return error",
		Argument:    optBucket(""),
		ExpectedErr: cli.ErrInvalidBucketName,
	},
	{
		Name:        "optEndpoint",
		Argument:    command.NewArguments(optEndpoint("endpoint"), optEndpoint("")),
		ExpectedCLI: []string{"cmd", "--endpoint=endpoint"},
	},
	{
		Name:        "optPrefix",
		Argument:    command.NewArguments(optPrefix("prefix"), optPrefix("")),
		ExpectedCLI: []string{"cmd", "--prefix=prefix", "--prefix="},
	},
	{
		Name:        "optDisableTLS",
		Argument:    command.NewArguments(optDisableTLS(true), optDisableTLS(false)),
		ExpectedCLI: []string{"cmd", "--disable-tls"},
	},
	{
		Name:        "optDisableTLSVerify",
		Argument:    command.NewArguments(optDisableTLSVerify(true), optDisableTLSVerify(false)),
		ExpectedCLI: []string{"cmd", "--disable-tls-verification"},
	},
}})
