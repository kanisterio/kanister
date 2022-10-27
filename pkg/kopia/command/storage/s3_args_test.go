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
	"time"

	"gopkg.in/check.v1"
)

func (s *StorageUtilsSuite) TestS3ArgsUtil(c *check.C) {
	artifactPrefix := "dir/sub-dir"
	for _, tc := range []struct {
		location        map[string]string
		expectedCommand string
	}{
		{
			location: map[string]string{
				bucketKey:        "test-bucket",
				prefixKey:        "test-prefix",
				regionKey:        "test-region",
				skipSSLVerifyKey: "true",
			},
			expectedCommand: fmt.Sprint(s3SubCommand,
				fmt.Sprintf(" %s=%s ", s3BucketFlag, "test-bucket"),
				fmt.Sprintf("%s=%s ", s3PrefixFlag, fmt.Sprintf("test-prefix/%s/", artifactPrefix)),
				s3DisableTLSVerifyFlag,
				fmt.Sprintf(" %s=test-region", s3RegionFlag),
			),
		},
		{
			location: map[string]string{
				bucketKey:   "test-bucket",
				prefixKey:   "test-prefix",
				endpointKey: "https://test.test:9000/",
			},
			expectedCommand: fmt.Sprint("s3 ",
				fmt.Sprintf("%s=%s ", s3BucketFlag, "test-bucket"),
				fmt.Sprintf("%s=%s ", s3EndpointFlag, "test.test:9000"),
				fmt.Sprintf("%s=%s", s3PrefixFlag, fmt.Sprintf("test-prefix/%s/", artifactPrefix))),
		},
		{
			location: map[string]string{
				bucketKey:   "test-bucket",
				prefixKey:   "test-prefix",
				endpointKey: "http://test.test:9000",
			},
			expectedCommand: fmt.Sprint("s3 ",
				fmt.Sprintf("%s=%s ", s3BucketFlag, "test-bucket"),
				fmt.Sprintf("%s=%s --disable-tls ", s3EndpointFlag, "test.test:9000"),
				fmt.Sprintf("%s=%s", s3PrefixFlag, fmt.Sprintf("test-prefix/%s/", artifactPrefix))),
		},
	} {
		args := kopiaS3Args(tc.location, time.Duration(30*time.Minute), artifactPrefix)
		c.Assert(args.String(), check.Equals, tc.expectedCommand)
	}
}
