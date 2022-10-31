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
)

func (s *StorageUtilsSuite) TestGCSArgsUtil(c *check.C) {
	locSecret := map[string]string{
		prefixKey: "test-prefix",
		bucketKey: "test-bucket",
	}
	artifactPrefix := "dir/sub-dir"
	cmd := kopiaGCSArgs(locSecret, artifactPrefix)
	c.Assert(cmd.String(), check.Equals, fmt.Sprint(
		gcsSubCommand,
		fmt.Sprintf(" --%s=%s", bucketKey, locSecret[bucketKey]),
		fmt.Sprintf(" %s=/tmp/creds.txt", credentialsFileFlag),
		fmt.Sprintf(" --%s=%s/%s/", prefixKey, locSecret[prefixKey], artifactPrefix),
	))
}
