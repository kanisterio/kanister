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

func (s *StorageUtilsSuite) TestStorageArgsUtil(c *check.C) {
	for _, tc := range []struct {
		params *StorageCommandParams
		check.Checker
		expectedCmd string
	}{
		{
			params: &StorageCommandParams{
				Location: map[string][]byte{
					bucketKey:        []byte("test-bucket"),
					prefixKey:        []byte("test-prefix"),
					regionKey:        []byte("test-region"),
					skipSSLVerifyKey: []byte("true"),
					typeKey:          []byte("s3"),
				},
				RepoPathPrefix: "dir/subdir/",
			},
			Checker: check.IsNil,
			expectedCmd: fmt.Sprint(
				s3SubCommand,
				fmt.Sprintf(" %s=test-bucket", s3BucketFlag),
				fmt.Sprintf(" %s=test-prefix/dir/subdir/ %s", s3PrefixFlag, s3DisableTLSVerifyFlag),
				fmt.Sprintf(" %s=test-region", s3RegionFlag),
			),
		},
		{
			params: &StorageCommandParams{
				Location: map[string][]byte{
					prefixKey: []byte("test-prefix"),
					typeKey:   []byte("filestore"),
				},
				RepoPathPrefix: "dir/subdir",
			},
			Checker: check.IsNil,
			expectedCmd: fmt.Sprint(
				filesystemSubCommand,
				fmt.Sprintf(" %s=/mnt/data/test-prefix/dir/subdir/", pathFlag),
			),
		},
		{
			params: &StorageCommandParams{
				Location: map[string][]byte{
					prefixKey: []byte("test-prefix"),
					bucketKey: []byte("test-bucket"),
					typeKey:   []byte("gcs"),
				},
				RepoPathPrefix: "dir/subdir",
			},
			Checker: check.IsNil,
			expectedCmd: fmt.Sprint(
				gcsSubCommand,
				fmt.Sprintf(" %s=test-bucket", gcsBucketFlag),
				fmt.Sprintf(" %s=/tmp/creds.txt", credentialsFileFlag),
				fmt.Sprintf(" %s=test-prefix/dir/subdir/", gcsPrefixFlag),
			),
		},
		{
			params: &StorageCommandParams{
				Location: map[string][]byte{
					bucketKey: []byte("test-bucket"),
					prefixKey: []byte("test-prefix"),
					typeKey:   []byte("azure"),
				},
				RepoPathPrefix: "dir/subdir",
			},
			Checker: check.IsNil,
			expectedCmd: fmt.Sprint(
				azureSubCommand,
				fmt.Sprintf(" %s=test-bucket", azureContainerFlag),
				fmt.Sprintf(" %s=test-prefix/dir/subdir/", azurePrefixFlag),
			),
		},
		{
			params: &StorageCommandParams{
				Location: map[string][]byte{
					typeKey: []byte("random-type"),
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
