// Copyright 2023 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package secrets

import (
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"

	"github.com/kanisterio/kanister/pkg/objectstore"
	secerrors "github.com/kanisterio/kanister/pkg/secrets/errors"
)

const (
	// AzureSecretType represents the secret type for Azure credentials.
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
func ValidateAzureCredentials(secret *corev1.Secret) error {
	if string(secret.Type) != AzureSecretType {
		return errors.Wrapf(secerrors.ErrValidate, secerrors.IncompatibleSecretTypeErrorMsg, AzureSecretType, secret.Namespace, secret.Name)
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
func ExtractAzureCredentials(secret *corev1.Secret) (*objectstore.SecretAzure, error) {
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
