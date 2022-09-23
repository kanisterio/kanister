package repository

import (
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/kanisterio/kanister/pkg/logsafe"
	"github.com/kanisterio/kanister/pkg/secrets"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
)

const (
	azureSubCommand         = "azure"
	azureContainerFlag      = "--container"
	azurePrefixFlag         = "--prefix"
	azureStorageAccountFlag = "--storage-account"
	azureStorageKeyFlag     = "--storage-key"
	azureStorageDomainFlag  = "--storage-domain"
)

func kopiaAzureArgs(locationSecret, locationCredSecret *v1.Secret, artifactPrefix string) (logsafe.Cmd, error) {
	artifactPrefix = GenerateFullRepoPath(prefix(locationSecret), artifactPrefix)

	args := logsafe.NewLoggable(azureSubCommand)
	args = args.AppendLoggableKV(azureContainerFlag, bucketName(locationSecret))
	args = args.AppendLoggableKV(azurePrefixFlag, artifactPrefix)

	credArgs, err := kopiaAzureCredentialArgs(locationCredSecret)
	if err != nil {
		return nil, err
	}

	return args.Combine(credArgs), nil
}

func kopiaAzureCredentialArgs(locationSecret *v1.Secret) (logsafe.Cmd, error) {
	azureSecret, err := secrets.ExtractAzureCredentials(locationSecret)
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
