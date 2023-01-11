// Copyright 2022 The Kanister Authors.
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

package command

import (
	"strings"
	"testing"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func TestKopiaCommandWrappers(t *testing.T) { TestingT(t) }

type KopiaBlobTestSuite struct{}

var _ = Suite(&KopiaBlobTestSuite{})

func (kBlob *KopiaBlobTestSuite) TestBlobCommands(c *C) {
	commandArgs := &CommandArgs{
		RepoPassword:   "encr-key",
		ConfigFilePath: "path/kopia.config",
		LogDirectory:   "cache/log",
	}

	for _, tc := range []struct {
		f           func() []string
		expectedLog string
	}{
		{
			f: func() []string {
				args := BlobListCommandArgs{
					CommandArgs: commandArgs,
				}
				return BlobList(args)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=encr-key blob list",
		},
		{
			f: func() []string {
				args := BlobStatsCommandArgs{
					CommandArgs: commandArgs,
				}
				return BlobStats(args)
			},
			expectedLog: "kopia --log-level=error --config-file=path/kopia.config --log-dir=cache/log --password=encr-key blob stats --raw",
		},
	} {
		cmd := strings.Join(tc.f(), " ")
		c.Assert(cmd, Equals, tc.expectedLog)
	}
}
