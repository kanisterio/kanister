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
	"time"

	"gopkg.in/check.v1"
)

func (s *StorageUtilsSuite) TestStorageArgsUtil(c *check.C) {
	for _, tc := range []struct {
		params *StorageCommandParams
		check.Checker
		expectedCmd string
	}{
		{
			params: &StorageCommandParams{
				Location: map[string]string{
					bucketKey:        "test-bucket",
					prefixKey:        "test-prefix",
					regionKey:        "test-region",
					skipSSLVerifyKey: "true",
					typeKey:          "s3",
				},
				AssumeRoleDuration: time.Duration(30 * time.Minute),
				RepoPathPrefix:     "dir/subdir/",
			},
			Checker:     check.IsNil,
			expectedCmd: "s3 --bucket=test-bucket --prefix=test-prefix/dir/subdir/ --disable-tls-verification --region=test-region",
		},
		{
			params: &StorageCommandParams{
				Location: map[string]string{
					prefixKey: "test-prefix",
					typeKey:   "filestore",
				},
				RepoPathPrefix: "dir/subdir",
			},
			Checker:     check.IsNil,
			expectedCmd: "filesystem --path=/mnt/data/test-prefix/dir/subdir/",
		},
		{
			params: &StorageCommandParams{
				Location: map[string]string{
					prefixKey: "test-prefix",
					bucketKey: "test-bucket",
					typeKey:   "gcs",
				},
				RepoPathPrefix: "dir/subdir",
			},
			Checker:     check.IsNil,
			expectedCmd: "gcs --bucket=test-bucket --credentials-file=/tmp/creds.txt --prefix=test-prefix/dir/subdir/",
		},
		{
			params: &StorageCommandParams{
				Location: map[string]string{
					bucketKey: "test-bucket",
					prefixKey: "test-prefix",
					typeKey:   "azure",
				},
				RepoPathPrefix: "dir/subdir",
			},
			Checker:     check.IsNil,
			expectedCmd: "azure --container=test-bucket --prefix=test-prefix/dir/subdir/",
		},
		{
			params: &StorageCommandParams{
				Location: map[string]string{
					typeKey: "random-type",
				},
			},
			Checker: check.NotNil,
		},
	} {
		cmd, err := KopiaStorageArgs(tc.params)
		c.Assert(err, tc.Checker)
		if tc.Checker == check.IsNil {
			c.Assert(cmd.String(), check.Equals, tc.expectedCmd)
		}
	}
}
