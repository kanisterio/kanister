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

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
	"gopkg.in/check.v1"
)

func TestStorageAzureFlags(t *testing.T) { check.TestingT(t) }

var _ = check.Suite(test.NewFlagSuite([]test.FlagTest{
	{
		Name: "Empty AzureCountainer should not generate a flag",
		Flag: Countainer(""),
	},
	{
		Name:        "AzureCountainer with value should generate a flag with the given value",
		Flag:        Countainer("container"),
		ExpectedCLI: []string{"--container=container"},
	},
	{
		Name: "Empty Prefix should not generate a flag",
		Flag: Prefix(""),
	},
	{
		Name:        "Prefix with value should generate a flag with the given value",
		Flag:        Prefix("prefix"),
		ExpectedCLI: []string{"--prefix=prefix"},
	},
}))
