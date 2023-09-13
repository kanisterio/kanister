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

func (s *StorageUtilsSuite) TestGCSArgsUtil(c *check.C) {
	locSecret := map[string][]byte{
		repositoryserver.PrefixKey: []byte("test-prefix"),
		repositoryserver.BucketKey: []byte("test-bucket"),
	}
	repoPathPrefix := "dir/sub-dir"
	cmd := gcsArgs(locSecret, repoPathPrefix)
	c.Assert(cmd.String(), check.Equals, fmt.Sprint(
		gcsSubCommand,
		fmt.Sprintf(" --%s=%s", repositoryserver.BucketKey, locSecret[repositoryserver.BucketKey]),
		fmt.Sprintf(" %s=/tmp/creds.txt", credentialsFileFlag),
		fmt.Sprintf(" --%s=%s/%s/", repositoryserver.PrefixKey, locSecret[repositoryserver.PrefixKey], repoPathPrefix),
	))
}
