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

package storage

import (
	"fmt"

	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/secrets/repositoryserver"
)

func (s *StorageUtilsSuite) TestAzureArgsUtil(c *check.C) {
	repoPathPrefix := "dir/sub-dir"
	for _, tc := range []struct {
		location        map[string][]byte
		expectedCommand string
	}{
		{
			location: map[string][]byte{
				repositoryserver.BucketKey: []byte("test-bucket"),
				repositoryserver.PrefixKey: []byte("test-prefix"),
			},
			expectedCommand: fmt.Sprint(azureSubCommand,
				fmt.Sprintf(" %s=%s ", azureContainerFlag, "test-bucket"),
				fmt.Sprintf("%s=%s", prefixFlag, fmt.Sprintf("test-prefix/%s/", repoPathPrefix)),
			),
		},
	} {
		cmd := azureArgs(tc.location, repoPathPrefix)
		c.Assert(cmd.String(), check.Equals, tc.expectedCommand)
	}
}
