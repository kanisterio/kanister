package secrets

import (
	"github.com/kanisterio/kanister/pkg/objectstore"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
)

const (
	AzureSecretType string = "secrets.kanister.io/azure"

	// AzureStorageAccountID is the config map key for Azure storage account id data
	AzureStorageAccountID string = "azure_storage_account_id"
	// AzureStorageAccountKey is the config map key for Azures storage account key data
	AzureStorageAccountKey string = "azure_storage_key"
	// AzureStorageEnvironment is the environment for Azures storage account
	AzureStorageEnvironment string = "azure_storage_environment"
)

// ValidateAzureCredentials validates secret has all necessary information
// for Azure credentials. It also checks the secret doesn't have unnecessary
// information.
//
// Required fields:
// - azure_storage_account_id
// - azure_storage_key
//
// Optional field:
// - azure_storage_environment
func ValidateAzureCredentials(secret *v1.Secret) error {
	if string(secret.Type) != AzureSecretType {
		return errors.New("Secret is not Azure secret")
	}
	count := 0
	if _, ok := secret.Data[AzureStorageAccountID]; ok {
		count++
	}
	if _, ok := secret.Data[AzureStorageAccountKey]; ok {
		count++
	}
	if _, ok := secret.Data[AzureStorageEnvironment]; ok {
		count++
	}
	if len(secret.Data) > count {
		return errors.New("Secret has an unknown field")
	}
	return nil
}

// ExtractAzureCredentials extracts Azure credential values from the given secret.
//
// Extracted values from the secrets are:
// - azure_storage_account_id (required)
// - azure_storage_key (required)
// - azure_storage_environment (optional)
//
// If the type of the secret is not "secrets.kanister.io/azure", it returns an error.
// If the required types are not available in the secrets, it returns an error.
func ExtractAzureCredentials(secret *v1.Secret) (*objectstore.SecretAzure, error) {
	if err := ValidateAzureCredentials(secret); err != nil {
		return nil, err
	}
	azSecret := &objectstore.SecretAzure{}
	if saID, ok := secret.Data[AzureStorageAccountID]; ok {
		azSecret.StorageAccount = string(saID)
	}
	if saKey, ok := secret.Data[AzureStorageAccountKey]; ok {
		azSecret.StorageKey = string(saKey)
	}
	if envName, ok := secret.Data[AzureStorageEnvironment]; ok {
		azSecret.EnvironmentName = string(envName)
	}
	if azSecret.StorageAccount == "" || azSecret.StorageKey == "" {
		return nil, errors.New("Azure secret is missing storage account ID or storage key")
	}
	return azSecret, nil
}
