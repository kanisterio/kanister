package storage

import (
	"fmt"
	"time"

	"gopkg.in/check.v1"
)

func (s *StorageUtilsSuite) TestS3ArgsUtil(c *check.C) {
	artifactPrefix := "dir/sub-dir"
	for _, tc := range []struct {
		location map[string]string
		check.Checker
		expectedCommand string
	}{
		{
			location: map[string]string{
				bucketKey:        "test-bucket",
				prefixKey:        "test-prefix",
				regionKey:        "test-region",
				skipSSLVerifyKey: "true",
			},
			Checker: check.IsNil,
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
			Checker: check.IsNil,
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
			Checker: check.IsNil,
			expectedCommand: fmt.Sprint("s3 ",
				fmt.Sprintf("%s=%s ", s3BucketFlag, "test-bucket"),
				fmt.Sprintf("%s=%s --disable-tls ", s3EndpointFlag, "test.test:9000"),
				fmt.Sprintf("%s=%s", s3PrefixFlag, fmt.Sprintf("test-prefix/%s/", artifactPrefix))),
		},
	} {
		args, err := kopiaS3Args(tc.location, time.Duration(30*time.Minute), artifactPrefix)
		c.Assert(err, tc.Checker)
		c.Assert(args, check.Not(check.Equals), tc.Checker)
		if tc.Checker == check.IsNil {
			c.Assert(args.String(), check.Equals, tc.expectedCommand)
		}
	}
}
