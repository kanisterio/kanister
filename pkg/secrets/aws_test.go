package secrets

import (
	"context"

	"github.com/aws/aws-sdk-go/aws/credentials"
	. "gopkg.in/check.v1"
	v1 "k8s.io/api/core/v1"

	"github.com/kanisterio/kanister/pkg/config"
)

type AWSSecretSuite struct{}

var _ = Suite(&AWSSecretSuite{})

func (s *AWSSecretSuite) TestExtractAWSCredentials(c *C) {
	tcs := []struct {
		secret     *v1.Secret
		expected   *credentials.Value
		errChecker Checker
	}{
		{
			secret: &v1.Secret{
				Type: v1.SecretType(AWSSecretType),
				Data: map[string][]byte{
					AWSAccessKeyID:     []byte("key_id"),
					AWSSecretAccessKey: []byte("secret_key"),
				},
			},
			expected: &credentials.Value{
				AccessKeyID:     "key_id",
				SecretAccessKey: "secret_key",
				ProviderName:    credentials.StaticProviderName,
			},
			errChecker: IsNil,
		},
		{
			secret: &v1.Secret{
				Type: "Opaque",
			},
			expected:   nil,
			errChecker: NotNil,
		},
		{
			secret: &v1.Secret{
				Type: v1.SecretType(AWSSecretType),
				Data: map[string][]byte{
					AWSSecretAccessKey: []byte("secret_key"),
				},
			},
			expected:   nil,
			errChecker: NotNil,
		},
		{
			secret: &v1.Secret{
				Type: v1.SecretType(AWSSecretType),
				Data: map[string][]byte{
					AWSAccessKeyID: []byte("key_id"),
				},
			},
			expected:   nil,
			errChecker: NotNil,
		},
		{
			secret: &v1.Secret{
				Type: v1.SecretType(AWSSecretType),
				Data: map[string][]byte{
					AWSAccessKeyID:     []byte("key_id"),
					AWSSecretAccessKey: []byte("secret_key"),
					"ExtraField":       []byte("extra_value"),
				},
			},
			expected:   nil,
			errChecker: NotNil,
		},
	}
	for _, tc := range tcs {
		creds, err := ExtractAWSCredentials(context.Background(), tc.secret)
		c.Check(creds, DeepEquals, tc.expected)
		c.Check(err, tc.errChecker)
	}
}

func (s *AWSSecretSuite) TestExtractAWSCredentialsWithSessionToken(c *C) {
	for _, tc := range []struct {
		secret *v1.Secret
		output Checker
	}{
		{
			secret: &v1.Secret{
				Type: v1.SecretType(AWSSecretType),
				Data: map[string][]byte{
					AWSAccessKeyID:     []byte(config.GetEnvOrSkip(c, "AWS_ACCESS_KEY_ID")),
					AWSSecretAccessKey: []byte(config.GetEnvOrSkip(c, "AWS_SECRET_ACCESS_KEY")),
					ConfigRole:         []byte(config.GetEnvOrSkip(c, "role")),
				},
			},
			output: IsNil,
		},
		{
			secret: &v1.Secret{
				Type: v1.SecretType(AWSSecretType),
				Data: map[string][]byte{
					AWSAccessKeyID:     []byte(config.GetEnvOrSkip(c, "AWS_ACCESS_KEY_ID")),
					AWSSecretAccessKey: []byte(config.GetEnvOrSkip(c, "AWS_SECRET_ACCESS_KEY")),
					ConfigRole:         []byte("arn:aws:iam::000000000000:role/test-fake-role"),
				},
			},
			output: NotNil,
		},
	} {
		_, err := ExtractAWSCredentials(context.Background(), tc.secret)
		c.Check(err, tc.output)
	}
}
