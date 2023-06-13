package secrets

import (
	v1 "k8s.io/api/core/v1"
)

const FilestoreSecretType string = "secrets.kanister.io/filestore"

// ValidateFileStoreCredentials validates secret has all necessary information
// for Filestore Credentials
func ValidateFileStoreCredentials(secret *v1.Secret) error {
	// Currently we dont need credentials for filestore hence
	// keeping the validation empty
	return nil
}
