package secrets

import (
	"github.com/kanisterio/kanister/pkg/objectstore"
	. "gopkg.in/check.v1"
	v1 "k8s.io/api/core/v1"
)

type AzureSecretSuite struct{}

var _ = Suite(&AzureSecretSuite{})

func (s *AzureSecretSuite) TestExtractAzureCredentials(c *C) {
	for i, tc := range []struct {
		secret     *v1.Secret
		expected   *objectstore.SecretAzure
		errChecker Checker
	}{
		{
			secret: &v1.Secret{
				Type: v1.SecretType(AzureSecretType),
				Data: map[string][]byte{
					AzureStorageAccountID:   []byte("key_id"),
					AzureStorageAccountKey:  []byte("secret_key"),
					AzureStorageEnvironment: []byte("env"),
				},
			},
			expected: &objectstore.SecretAzure{
				StorageAccount:  "key_id",
				StorageKey:      "secret_key",
				EnvironmentName: "env",
			},
			errChecker: IsNil,
		},
		{ // bad type
			secret: &v1.Secret{
				Type: v1.SecretType(AWSSecretType),
				Data: map[string][]byte{
					AzureStorageAccountID:   []byte("key_id"),
					AzureStorageAccountKey:  []byte("secret_key"),
					AzureStorageEnvironment: []byte("env"),
				},
			},
			expected:   nil,
			errChecker: NotNil,
		},
		{ // missing field
			secret: &v1.Secret{
				Type: v1.SecretType(AzureSecretType),
				Data: map[string][]byte{
					AzureStorageAccountID:   []byte("key_id"),
					AzureStorageEnvironment: []byte("env"),
				},
			},
			expected:   nil,
			errChecker: NotNil,
		},
		{ // additional field
			secret: &v1.Secret{
				Type: v1.SecretType(AzureSecretType),
				Data: map[string][]byte{
					AzureStorageAccountID:   []byte("key_id"),
					AzureStorageAccountKey:  []byte("secret_key"),
					AzureStorageEnvironment: []byte("env"),
					"bad field":             []byte("bad"),
				},
			},
			expected:   nil,
			errChecker: NotNil,
		},
	} {
		azsecret, err := ExtractAzureCredentials(tc.secret)
		c.Check(azsecret, DeepEquals, tc.expected, Commentf("test number: %d", i))
		c.Check(err, tc.errChecker)
	}
}
