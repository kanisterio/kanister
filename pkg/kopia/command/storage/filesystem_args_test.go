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

func (s *StorageUtilsSuite) TestFilesystemArgsUtil(c *check.C) {
	for _, tc := range []struct {
		prefix         string
		repoPathPrefix string
		expectedPath   string
	}{
		{
			prefix:         "",
			repoPathPrefix: "dir1/subdir/",
			expectedPath:   fmt.Sprintf("%s/dir1/subdir/", DefaultFSMountPath),
		},
		{
			prefix:         "test-prefix",
			repoPathPrefix: "dir1/subdir/",
			expectedPath:   fmt.Sprintf("%s/test-prefix/dir1/subdir/", DefaultFSMountPath),
		},
	} {
		sec := map[string][]byte{
			repositoryserver.PrefixKey: []byte(tc.prefix),
		}
		args := filesystemArgs(sec, tc.repoPathPrefix)
		expectedValue := fmt.Sprint(
			filesystemSubCommand,
			fmt.Sprintf(" %s=%s", pathFlag, tc.expectedPath))
		c.Assert(args.String(), check.Equals, expectedValue)
	}
}
