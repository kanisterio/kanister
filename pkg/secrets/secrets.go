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

	"github.com/kanisterio/kanister/pkg/secrets/repositoryserver"
)

// ValidateCredentials returns error if secret is failed at validation.
// Currently supports following:
// - AWS typed secret with required AWS secret fields.
func ValidateCredentials(secret *corev1.Secret) error {
	if secret == nil {
		return errors.New("Nil secret")
	}
	switch string(secret.Type) {
	case AWSSecretType:
		return ValidateAWSCredentials(secret)
	case AzureSecretType:
		return ValidateAzureCredentials(secret)
	case GCPSecretType:
		return ValidateGCPCredentials(secret)
	default:
		return errors.Errorf("Unsupported type '%s' for secret '%s:%s'", string(secret.Type), secret.Namespace, secret.Name)
	}
}

func getLocationType(secret *corev1.Secret) (repositoryserver.RepositoryServerSecret, error) {
	var locationType []byte
	var ok bool
	if secret == nil {
		return nil, errors.New("Secret is Nil")
	}

	if locationType, ok = (secret.Data[LocationTypeKey]); !ok {
		return nil, errors.Errorf("secret '%s:%s' does not have required field %s", secret.Namespace, secret.Name, LocationTypeKey)
	}

	switch LocType(string(locationType)) {
	case LocTypeS3:
		return repositoryserver.NewAWSLocation(secret), nil
	case LocTypeAzure:
		return repositoryserver.NewAzureLocation(secret), nil
	case LocTypeGCS:
		return repositoryserver.NewGCPLocation(secret), nil
	default:
		return nil, errors.Errorf("Unsupported location type '%s' for secret '%s:%s'", locationType, secret.Namespace, secret.Name)
	}
}

func ValidateRepositoryServerSecret(repositoryServerSecret *corev1.Secret) error {
	var secret repositoryserver.RepositoryServerSecret
	var err error

	switch repositoryServerSecret.Type {
	case Location:
		secret, err = getLocationType(repositoryServerSecret)
		if err != nil {
			return err
		}
	case RepositoryPassword:
		secret = repositoryserver.NewRepoPassword(repositoryServerSecret)
	case RepositoryServerAdminCredentials:
		secret = repositoryserver.NewRepositoryServerAdminCredentials(repositoryServerSecret)
	default:
		return ValidateCredentials(repositoryServerSecret)
	}
	return secret.Validate()
}
