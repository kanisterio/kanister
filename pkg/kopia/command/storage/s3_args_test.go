package storage

import (
	"fmt"
	"time"

	"gopkg.in/check.v1"
	v1 "k8s.io/api/core/v1"

	"github.com/kanisterio/kanister/pkg/secrets"
)

func (s *StorageUtilsSuite) TestS3ArgsUtil(c *check.C) {
	artifactPrefix := "dir/sub-dir"
	for _, tc := range []struct {
		locSec  *v1.Secret
		credSec *v1.Secret
		check.Checker
		expectedCommand string
	}{
		{
			locSec: &v1.Secret{
				StringData: map[string]string{
					bucketKey:        "test-bucket",
					prefixKey:        "test-prefix",
					regionKey:        "test-region",
					skipSSLVerifyKey: "true",
				},
			},
			credSec: &v1.Secret{
				Type: v1.SecretType(secrets.AWSSecretType),
				Data: map[string][]byte{
					secrets.AWSAccessKeyID:     []byte("test-access-key-id"),
					secrets.AWSSecretAccessKey: []byte("test-secret-access-key"),
				},
			},
			Checker: check.IsNil,
			expectedCommand: fmt.Sprint(s3SubCommand,
				fmt.Sprintf(" %s=%s ", s3BucketFlag, "test-bucket"),
				fmt.Sprintf("%s=<****> ", s3AccessKeyFlag),
				fmt.Sprintf("%s=<****> ", s3SecretAccessKeyFlag),
				fmt.Sprintf("%s=%s ", s3PrefixFlag, fmt.Sprintf("test-prefix/%s/", artifactPrefix)),
				s3DisableTLSVerifyFlag,
				fmt.Sprintf(" %s=test-region", s3RegionFlag),
			),
		},
		{
			locSec: &v1.Secret{
				StringData: map[string]string{
					bucketKey:   "test-bucket",
					prefixKey:   "test-prefix",
					endpointKey: "https://test.test:9000/",
				},
			},
			credSec: &v1.Secret{
				Type: v1.SecretType(secrets.AWSSecretType),
				Data: map[string][]byte{
					secrets.AWSAccessKeyID:     []byte("test-access-key-id"),
					secrets.AWSSecretAccessKey: []byte("test-secret-access-key"),
				},
			},
			Checker: check.IsNil,
			expectedCommand: fmt.Sprint("s3 ",
				fmt.Sprintf("%s=%s ", s3BucketFlag, "test-bucket"),
				fmt.Sprintf("%s=%s ", s3EndpointFlag, "test.test:9000"),
				fmt.Sprintf("%s=<****> ", s3AccessKeyFlag),
				fmt.Sprintf("%s=<****> ", s3SecretAccessKeyFlag),
				fmt.Sprintf("%s=%s", s3PrefixFlag, fmt.Sprintf("test-prefix/%s/", artifactPrefix))),
		},
		{
			locSec: &v1.Secret{
				StringData: map[string]string{
					bucketKey:   "test-bucket",
					prefixKey:   "test-prefix",
					endpointKey: "http://test.test:9000",
				},
			},
			credSec: &v1.Secret{
				Type: v1.SecretType(secrets.AWSSecretType),
				Data: map[string][]byte{
					secrets.AWSAccessKeyID:     []byte("test-access-key-id"),
					secrets.AWSSecretAccessKey: []byte("test-secret-access-key"),
				},
			},
			Checker: check.IsNil,
			expectedCommand: fmt.Sprint("s3 ",
				fmt.Sprintf("%s=%s ", s3BucketFlag, "test-bucket"),
				fmt.Sprintf("%s=%s --disable-tls ", s3EndpointFlag, "test.test:9000"),
				fmt.Sprintf("%s=<****> ", s3AccessKeyFlag),
				fmt.Sprintf("%s=<****> ", s3SecretAccessKeyFlag),
				fmt.Sprintf("%s=%s", s3PrefixFlag, fmt.Sprintf("test-prefix/%s/", artifactPrefix))),
		},
		{
			locSec: &v1.Secret{
				StringData: map[string]string{
					bucketKey: "test-bucket",
					prefixKey: "test-prefix",
				},
			},
			credSec: &v1.Secret{
				Data: map[string][]byte{
					secrets.AWSAccessKeyID:     []byte("test-access-key-id"),
					secrets.AWSSecretAccessKey: []byte("test-secret-access-key"),
				},
			},
			Checker: check.NotNil,
		},
	} {
		args, err := kopiaS3Args(tc.locSec, tc.credSec, time.Duration(30*time.Minute), artifactPrefix)
		c.Assert(err, tc.Checker)
		c.Assert(args, check.Not(check.Equals), tc.Checker)
		if tc.Checker == check.IsNil {
			c.Assert(args.String(), check.Equals, tc.expectedCommand)
		}
	}
}
