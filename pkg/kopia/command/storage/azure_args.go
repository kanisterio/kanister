package storage

import (
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/kanisterio/kanister/pkg/logsafe"
	"github.com/kanisterio/kanister/pkg/objectstore"
	"github.com/kanisterio/kanister/pkg/secrets"
	"github.com/pkg/errors"
)

const (
	azureSubCommand         = "azure"
	azureContainerFlag      = "--container"
	azurePrefixFlag         = "--prefix"
	azureStorageAccountFlag = "--storage-account"
	azureStorageKeyFlag     = "--storage-key"
	azureStorageDomainFlag  = "--storage-domain"
)

func kopiaAzureArgs(location map[string]string, credentials map[string]string, artifactPrefix string) (logsafe.Cmd, error) {
	artifactPrefix = GenerateFullRepoPath(prefix(location), artifactPrefix)

	args := logsafe.NewLoggable(azureSubCommand)
	args = args.AppendLoggableKV(azureContainerFlag, bucketName(location))
	args = args.AppendLoggableKV(azurePrefixFlag, artifactPrefix)

	credArgs, err := kopiaAzureCredentialArgs(credentials)
	if err != nil {
		return nil, err
	}

	return args.Combine(credArgs), nil
}

func kopiaAzureCredentialArgs(credentials map[string]string) (logsafe.Cmd, error) {
	azureSecret, err := extractAzureCredentials(credentials)
	if err != nil {
		return nil, err
	}
	storageAccount := azureSecret.StorageAccount
	storageKey := azureSecret.StorageKey
	storageEnv := azureSecret.EnvironmentName
	args := logsafe.Cmd{}
	args = args.AppendRedactedKV(azureStorageAccountFlag, storageAccount)
	args = args.AppendRedactedKV(azureStorageKeyFlag, storageKey)
	if storageEnv != "" {
		env, err := azure.EnvironmentFromName(storageEnv)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to get azure environment from name: %s", storageEnv)
		}
		blobDomain := "blob." + env.StorageEndpointSuffix
		args = args.AppendLoggableKV(azureStorageDomainFlag, blobDomain)
	}
	return args, nil
}

func extractAzureCredentials(credentials map[string]string) (*objectstore.SecretAzure, error) {
	if err := validateAzureCredentials(credentials); err != nil {
		return nil, err
	}
	azSecret := &objectstore.SecretAzure{}
	if saID, ok := credentials[secrets.AzureStorageAccountID]; ok {
		azSecret.StorageAccount = string(saID)
	}
	if saKey, ok := credentials[secrets.AzureStorageAccountKey]; ok {
		azSecret.StorageKey = string(saKey)
	}
	if envName, ok := credentials[secrets.AzureStorageEnvironment]; ok {
		azSecret.EnvironmentName = string(envName)
	}
	if azSecret.StorageAccount == "" || azSecret.StorageKey == "" {
		return nil, errors.New("Azure secret is missing storage account ID or storage key")
	}
	return azSecret, nil
}

func validateAzureCredentials(credentials map[string]string) error {
	count := 0
	if _, ok := credentials[secrets.AzureStorageAccountID]; ok {
		count++
	}
	if _, ok := credentials[secrets.AzureStorageAccountKey]; ok {
		count++
	}
	if _, ok := credentials[secrets.AzureStorageEnvironment]; ok {
		count++
	}
	if len(credentials) > count {
		return errors.New("Secret has an unknown field")
	}
	return nil
}
