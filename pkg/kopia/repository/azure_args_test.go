package repository

import (
	"fmt"

	"gopkg.in/check.v1"
	v1 "k8s.io/api/core/v1"

	"github.com/kanisterio/kanister/pkg/secrets"
)

func (s *RepositoryUtilsSuite) TestAzureArgsUtil(c *check.C) {
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
					bucketKey: "test-bucket",
					prefixKey: "test-prefix",
				},
			},
			credSec: &v1.Secret{
				Type: v1.SecretType(secrets.AzureSecretType),
				Data: map[string][]byte{
					secrets.AzureStorageAccountID:   []byte("test-storage-account-id"),
					secrets.AzureStorageAccountKey:  []byte("test-storage-account-key"),
					secrets.AzureStorageEnvironment: []byte("AZURECLOUD"),
				},
			},
			Checker: check.IsNil,
			expectedCommand: fmt.Sprint(azureSubCommand,
				fmt.Sprintf(" %s=%s ", azureContainerFlag, "test-bucket"),
				fmt.Sprintf("%s=%s ", azurePrefixFlag, fmt.Sprintf("test-prefix/%s/", artifactPrefix)),
				fmt.Sprintf("%s=<****> ", azureStorageAccountFlag),
				fmt.Sprintf("%s=<****> ", azureStorageKeyFlag),
				fmt.Sprintf("%s=blob.core.windows.net", azureStorageDomainFlag),
			),
		},
		{
			locSec: &v1.Secret{
				StringData: map[string]string{
					bucketKey: "test-bucket",
					prefixKey: "test-prefix",
				},
			},
			credSec: &v1.Secret{
				Type: v1.SecretType(secrets.AzureSecretType),
				Data: map[string][]byte{
					secrets.AzureStorageAccountID:   []byte("test-storage-account-id"),
					secrets.AzureStorageAccountKey:  []byte("test-storage-account-key"),
					secrets.AzureStorageEnvironment: []byte("RANDOM"),
				},
			},
			Checker: check.NotNil,
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
					secrets.AzureStorageAccountID:   []byte("test-storage-account-id"),
					secrets.AzureStorageAccountKey:  []byte("test-storage-account-key"),
					secrets.AzureStorageEnvironment: []byte("AZURECLOUD"),
				},
			},
			Checker: check.NotNil,
		},
	} {
		cmd, err := kopiaAzureArgs(tc.locSec, tc.credSec, artifactPrefix)
		c.Assert(err, tc.Checker)
		if tc.Checker == check.IsNil {
			c.Assert(cmd.String(), check.Equals, tc.expectedCommand)
		}
	}
}
