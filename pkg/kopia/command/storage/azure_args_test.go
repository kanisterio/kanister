package storage

import (
	"fmt"

	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/secrets"
)

func (s *StorageUtilsSuite) TestAzureArgsUtil(c *check.C) {
	artifactPrefix := "dir/sub-dir"
	for _, tc := range []struct {
		location    map[string]string
		credentials map[string]string
		check.Checker
		expectedCommand string
	}{
		{
			location: map[string]string{
				bucketKey: "test-bucket",
				prefixKey: "test-prefix",
			},
			credentials: map[string]string{
				secrets.AzureStorageAccountID:   "test-storage-account-id",
				secrets.AzureStorageAccountKey:  "test-storage-account-key",
				secrets.AzureStorageEnvironment: "AZURECLOUD",
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
			location: map[string]string{
				bucketKey: "test-bucket",
				prefixKey: "test-prefix",
			},
			credentials: map[string]string{
				secrets.AzureStorageAccountID:   "test-storage-account-id",
				secrets.AzureStorageAccountKey:  "test-storage-account-key",
				secrets.AzureStorageEnvironment: "RANDOM",
			},
			Checker: check.NotNil,
		},
	} {
		cmd, err := kopiaAzureArgs(tc.location, tc.credentials, artifactPrefix)
		c.Assert(err, tc.Checker)
		if tc.Checker == check.IsNil {
			c.Assert(cmd.String(), check.Equals, tc.expectedCommand)
		}
	}
}
