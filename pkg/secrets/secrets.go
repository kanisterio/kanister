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
	"github.com/kanisterio/kanister/pkg/kopia/command/storage"
	"github.com/kanisterio/kanister/pkg/secrets/repositoryserver"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
)

const (
	// LocationSecretType represents the storage location secret type for kopia repository server
	LocationSecretType string = "secrets.kanister.io/storage-location"
	// LocationTypeKey represents the key used to define the location type in
	// the kopia repository server location secret
	LocationTypeKey string = "type"
	// RepositoryPasswordSecretType represents the kopia repository passowrd secret type
	RepositoryPasswordSecretType string = "secrets.kanister.io/kopia-repository/password"
	// RepositoryServerAdminCredentialsSecretType represents the kopia server admin credentials secret type
	RepositoryServerAdminCredentialsSecretType string = "secrets.kanister.io/kopia-repository/serveradmin"
)

// ValidateCredentials returns error if secret is failed at validation.
// Currently supports following:
// - AWS typed secret with required AWS secret fields.
func ValidateCredentials(secret *v1.Secret) error {
	if secret == nil {
		return errors.New("Nil secret")
	}
	switch string(secret.Type) {
	case AWSSecretType:
		return ValidateAWSCredentials(secret)
	case AzureSecretType:
		return ValidateAzureCredentials(secret)
	default:
		return errors.Errorf("Unsupported type '%s' for secret '%s:%s'", string(secret.Type), secret.Namespace, secret.Name)
	}
}

func getLocationType(secret *v1.Secret) (repositoryserver.RepositoryServerSecret, error) {
	var locationType []byte
	var ok bool
	if secret == nil {
		return nil, errors.New("Secret is Nil")
	}

	if locationType, ok = (secret.Data[LocationTypeKey]); !ok {
		return nil, errors.Errorf("secret '%s:%s' does not have required field %s", secret.Namespace, secret.Name, LocationTypeKey)
	}

	switch string(locationType) {
	case storage.LocTypeS3:
		return repositoryserver.NewAWSLocation(secret), nil
	case storage.LocTypeAzure:
		return repositoryserver.NewAzureLocation(secret), nil
	default:
		return nil, errors.Errorf("Unsupported location type '%s' for secret '%s:%s'", locationType, secret.Namespace, secret.Name)
	}
}

func ValidateRepositoryServerSecret(repositoryServerSecret *v1.Secret) error {
	var secret repositoryserver.RepositoryServerSecret
	var err error

	switch string(repositoryServerSecret.Type) {
	case LocationSecretType:
		secret, err = getLocationType(repositoryServerSecret)
		if err != nil {
			return err
		}
	case RepositoryPasswordSecretType:
		secret = repositoryserver.NewRepoPassword(repositoryServerSecret)
	case RepositoryServerAdminCredentialsSecretType:
		secret = repositoryserver.NewRepositoryServerAdminCredentials(repositoryServerSecret)
	default:
		return ValidateCredentials(repositoryServerSecret)
	}
	return secret.Validate()
}
