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
)

func TestAzureOptions(t *testing.T) { check.TestingT(t) }

var _ = check.Suite(&test.ArgumentSuite{Cmd: "cmd", Arguments: []test.ArgumentTest{
	{
		Name:        "optContainer",
		Argument:    optContainer("containername"),
		ExpectedCLI: []string{"cmd", "--container=containername"},
	},
	{
		Name:        "optContainer with empty containername should return error",
		Argument:    optContainer(""),
		ExpectedErr: cli.ErrInvalidContainerName,
	},
	{
		Name:        "optPrefix",
		Argument:    command.NewArguments(optPrefix("prefix"), optPrefix("")),
		ExpectedCLI: []string{"cmd", "--prefix=prefix", "--prefix="},
	},
}})
