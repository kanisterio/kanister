package storage

import (
	"github.com/kanisterio/kanister/pkg/logsafe"
)

const (
	azureSubCommand         = "azure"
	azureContainerFlag      = "--container"
	azurePrefixFlag         = "--prefix"
	azureStorageAccountFlag = "--storage-account"
	azureStorageKeyFlag     = "--storage-key"
	azureStorageDomainFlag  = "--storage-domain"
)

func kopiaAzureArgs(location map[string]string, artifactPrefix string) (logsafe.Cmd, error) {
	artifactPrefix = GenerateFullRepoPath(prefix(location), artifactPrefix)

	args := logsafe.NewLoggable(azureSubCommand)
	args = args.AppendLoggableKV(azureContainerFlag, bucketName(location))
	args = args.AppendLoggableKV(azurePrefixFlag, artifactPrefix)

	return args, nil
}
