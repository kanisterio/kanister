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

	secerrors "github.com/kanisterio/kanister/pkg/secrets/errors"
	reposerver "github.com/kanisterio/kanister/pkg/secrets/repositoryserver"
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
	case FilestoreSecretType:
		// returning nil currently since we
		// dont need credentials for file store
		return nil
	default:
		return errors.Errorf("Unsupported type '%s' for secret '%s:%s'", string(secret.Type), secret.Namespace, secret.Name)
	}
}

func getLocationSecret(secret *corev1.Secret) (reposerver.Secret, error) {
	var locationType []byte
	var ok bool
	if secret == nil {
		return nil, errors.New("Secret for kopia repository location is Nil")
	}

	if locationType, ok = (secret.Data[reposerver.TypeKey]); !ok {
		return nil, errors.Wrapf(secerrors.ErrValidate, secerrors.MissingRequiredFieldErrorMsg, reposerver.TypeKey, secret.Namespace, secret.Name)
	}

	switch reposerver.LocType(string(locationType)) {
	case reposerver.LocTypeS3:
		return reposerver.NewAWSLocation(secret), nil
	case reposerver.LocTypes3Compliant:
		return reposerver.NewS3CompliantLocation(secret), nil
	case reposerver.LocTypeAzure:
		return reposerver.NewAzureLocation(secret), nil
	case reposerver.LocTypeGCS:
		return reposerver.NewGCPLocation(secret), nil
	case reposerver.LocTypeFilestore:
		return reposerver.NewFileStoreLocation(secret), nil
	default:
		return nil, errors.Wrapf(secerrors.ErrValidate, secerrors.UnsupportedLocationTypeErrorMsg, locationType, secret.Namespace, secret.Name)
	}
}

func ValidateRepositoryServerSecret(repositoryServerSecret *corev1.Secret) error {
	var secret reposerver.Secret
	var err error

	switch repositoryServerSecret.Type {
	case reposerver.Location:
		secret, err = getLocationSecret(repositoryServerSecret)
		if err != nil {
			return err
		}
	case reposerver.RepositoryPasswordSecret:
		secret = reposerver.NewRepoPassword(repositoryServerSecret)
	case reposerver.AdminCredentialsSecret:
		secret = reposerver.NewRepositoryServerAdminCredentials(repositoryServerSecret)
	default:
		return ValidateCredentials(repositoryServerSecret)
	}
	return secret.Validate()
}
