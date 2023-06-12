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

func (s *StorageUtilsSuite) TestS3ArgsUtil(c *check.C) {
	repoPathPrefix := "dir/sub-dir"
	for _, tc := range []struct {
		location        map[string][]byte
		expectedCommand string
	}{
		{
			location: map[string][]byte{
				repositoryserver.BucketKey:        []byte("test-bucket"),
				repositoryserver.PrefixKey:        []byte("test-prefix"),
				repositoryserver.RegionKey:        []byte("test-region"),
				repositoryserver.SkipSSLVerifyKey: []byte("true"),
			},
			expectedCommand: fmt.Sprint(s3SubCommand,
				fmt.Sprintf(" %s=%s", bucketFlag, "test-bucket"),
				fmt.Sprintf(" %s=%s ", prefixFlag, fmt.Sprintf("test-prefix/%s/", repoPathPrefix)),
				s3DisableTLSVerifyFlag,
				fmt.Sprintf(" %s=test-region", s3RegionFlag),
			),
		},
		{
			location: map[string][]byte{
				repositoryserver.BucketKey:   []byte("test-bucket"),
				repositoryserver.PrefixKey:   []byte("test-prefix"),
				repositoryserver.EndpointKey: []byte("https://test.test:9000/"),
			},
			expectedCommand: fmt.Sprint(s3SubCommand,
				fmt.Sprintf(" %s=%s", bucketFlag, "test-bucket"),
				fmt.Sprintf(" %s=%s", s3EndpointFlag, "test.test:9000"),
				fmt.Sprintf(" %s=%s", prefixFlag, fmt.Sprintf("test-prefix/%s/", repoPathPrefix))),
		},
		{
			location: map[string][]byte{
				repositoryserver.BucketKey:   []byte("test-bucket"),
				repositoryserver.PrefixKey:   []byte("test-prefix"),
				repositoryserver.EndpointKey: []byte("http://test.test:9000"),
			},
			expectedCommand: fmt.Sprint(s3SubCommand,
				fmt.Sprintf(" %s=%s", bucketFlag, "test-bucket"),
				fmt.Sprintf(" %s=test.test:9000 %s", s3EndpointFlag, s3DisableTLSFlag),
				fmt.Sprintf(" %s=test-prefix/%s/", prefixFlag, repoPathPrefix)),
		},
	} {
		args := s3Args(tc.location, repoPathPrefix)
		c.Assert(args.String(), check.Equals, tc.expectedCommand)
	}
}

func (s *StorageUtilsSuite) TestResolveS3Endpoint(c *check.C) {
	for _, tc := range []struct {
		endpoint       string
		expectedOutput string
	}{
		{
			endpoint:       "http://example:8000",
			expectedOutput: "example:8000",
		},
		{
			endpoint:       "http://example:8000/",
			expectedOutput: "example:8000",
		},
		{
			endpoint:       "https://example:8000",
			expectedOutput: "example:8000",
		},
		{
			endpoint:       "https://example:8000/",
			expectedOutput: "example:8000",
		},
		{
			endpoint:       "example:8000",
			expectedOutput: "example:8000",
		},
		{
			endpoint:       "example",
			expectedOutput: "example",
		},
	} {
		op := ResolveS3Endpoint(tc.endpoint)
		c.Assert(op, check.Equals, tc.expectedOutput)
	}
}
