package storage

import (
	"time"

	"github.com/kanisterio/kanister/pkg/secrets"
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
				Credentials: map[string]string{
					secrets.AWSAccessKeyID:     "test-access-key-id",
					secrets.AWSSecretAccessKey: "test-secret-access-key",
				},
				AssumeRoleDuration: time.Duration(30 * time.Minute),
				RepoPathPrefix:     "dir/subdir/",
			},
			Checker:     check.IsNil,
			expectedCmd: "s3 --bucket=test-bucket --access-key=<****> --secret-access-key=<****> --prefix=test-prefix/dir/subdir/ --disable-tls-verification --region=test-region",
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
				Credentials: map[string]string{
					secrets.AzureStorageAccountID:   "test-storage-account-id",
					secrets.AzureStorageAccountKey:  "test-storage-account-key",
					secrets.AzureStorageEnvironment: "AZURECLOUD",
				},
				RepoPathPrefix: "dir/subdir",
			},
			Checker:     check.IsNil,
			expectedCmd: "azure --container=test-bucket --prefix=test-prefix/dir/subdir/ --storage-account=<****> --storage-key=<****> --storage-domain=blob.core.windows.net",
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
		cmd, err := KopiaBlobStoreArgs(tc.params)
		c.Assert(err, tc.Checker)
		if tc.Checker == check.IsNil {
			c.Assert(cmd.String(), check.Equals, tc.expectedCmd)
		}
	}
}
