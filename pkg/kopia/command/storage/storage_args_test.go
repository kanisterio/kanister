package storage

import (
	"time"

	"github.com/kanisterio/kanister/pkg/secrets"
	"gopkg.in/check.v1"
	v1 "k8s.io/api/core/v1"
)

func (s *StorageUtilsSuite) TestStorageArgsUtil(c *check.C) {
	for _, tc := range []struct {
		params *StorageCommandParams
		check.Checker
		expectedCmd string
	}{
		{
			params: &StorageCommandParams{
				LocationSecret: &v1.Secret{
					StringData: map[string]string{
						bucketKey:        "test-bucket",
						prefixKey:        "test-prefix",
						regionKey:        "test-region",
						skipSSLVerifyKey: "true",
						typeKey:          "s3",
					},
				},
				LocationCredSecret: &v1.Secret{
					Type: v1.SecretType(secrets.AWSSecretType),
					Data: map[string][]byte{
						secrets.AWSAccessKeyID:     []byte("test-access-key-id"),
						secrets.AWSSecretAccessKey: []byte("test-secret-access-key"),
					},
				},
				AssumeRoleDuration: time.Duration(30 * time.Minute),
				RepoPathPrefix:     "dir/subdir/",
			},
			Checker:     check.IsNil,
			expectedCmd: "s3 --bucket=test-bucket --access-key=<****> --secret-access-key=<****> --prefix=test-prefix/dir/subdir/ --disable-tls-verification --region=test-region",
		},
		{
			params: &StorageCommandParams{
				LocationSecret: &v1.Secret{
					StringData: map[string]string{
						prefixKey: "test-prefix",
						typeKey:   "filestore",
					},
				},
				RepoPathPrefix: "dir/subdir",
			},
			Checker:     check.IsNil,
			expectedCmd: "filesystem --path=/mnt/data/test-prefix/dir/subdir/",
		},
		{
			params: &StorageCommandParams{
				LocationSecret: &v1.Secret{
					StringData: map[string]string{
						prefixKey: "test-prefix",
						bucketKey: "test-bucket",
						typeKey:   "gcs",
					},
				},
				RepoPathPrefix: "dir/subdir",
			},
			Checker:     check.IsNil,
			expectedCmd: "gcs --bucket=test-bucket --credentials-file=/tmp/creds.txt --prefix=test-prefix/dir/subdir/",
		},
		{
			params: &StorageCommandParams{
				LocationSecret: &v1.Secret{
					StringData: map[string]string{
						bucketKey: "test-bucket",
						prefixKey: "test-prefix",
						typeKey:   "azure",
					},
				},
				LocationCredSecret: &v1.Secret{
					Type: v1.SecretType(secrets.AzureSecretType),
					Data: map[string][]byte{
						secrets.AzureStorageAccountID:   []byte("test-storage-account-id"),
						secrets.AzureStorageAccountKey:  []byte("test-storage-account-key"),
						secrets.AzureStorageEnvironment: []byte("AZURECLOUD"),
					},
				},
				RepoPathPrefix: "dir/subdir",
			},
			Checker:     check.IsNil,
			expectedCmd: "azure --container=test-bucket --prefix=test-prefix/dir/subdir/ --storage-account=<****> --storage-key=<****> --storage-domain=blob.core.windows.net",
		},
		{
			params: &StorageCommandParams{
				LocationSecret: &v1.Secret{
					StringData: map[string]string{
						typeKey: "random-type",
					},
				},
			},
			Checker: check.NotNil,
		},
	} {
		cmd, err := KopiaBlobStoreArgs(tc.params)
		c.Assert(err, tc.Checker)
		if tc.Checker == check.IsNil {
			c.Assert(cmd.String(), check.Equals, tc.expectedCmd)
		}
	}
}
