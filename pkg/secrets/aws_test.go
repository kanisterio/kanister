package secrets

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws/credentials"
	. "gopkg.in/check.v1"
	v1 "k8s.io/api/core/v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

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
			},
			errChecker: IsNil,
		},
		{
			secret: &v1.Secret{
				Type: v1.SecretType(AWSSecretType),
				Data: map[string][]byte{
					AWSAccessKeyID:     []byte("key_id"),
					AWSSecretAccessKey: []byte("secret_key"),
					AWSSessionToken:    []byte("session_token"),
				},
			},
			expected: &credentials.Value{
				AccessKeyID:     "key_id",
				SecretAccessKey: "secret_key",
				SessionToken:    "session_token",
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
	}
	for _, tc := range tcs {
		creds, err := ExtractAWSCredentials(tc.secret)
		c.Check(creds, DeepEquals, tc.expected)
		c.Check(err, tc.errChecker)
	}
}
