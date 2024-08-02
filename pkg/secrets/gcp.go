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
)

const (
	// GCPProjectID is the config map key for gcp project id data
	GCPProjectID string = "gcp_project_id"
	// GCPServiceKey is the config map key for gcp service key data
	GCPServiceKey string = "gcp_service_key"
	// GCPServerAccountJsonKey is the key for gcp service account json
	GCPServiceAccountJSONKey string = "service-account.json"

	// GCPSecretType represents the secret type for GCP credentials.
	GCPSecretType string = "secrets.kanister.io/gcp"
)

// ValidateGCPCredentials function is to verify the schema of GCP secrets
// that need to be provided for kopia commands
func ValidateGCPCredentials(secret *corev1.Secret) error {
	// Required fields for the secret are
	// - GCPProjectID
	// - GCPServiceAccountJSONKey
	if secret == nil {
		return errors.Wrapf(secerrors.ErrValidate, secerrors.NilSecretErrorMessage)
	}
	if string(secret.Type) != GCPSecretType {
		return errors.Wrapf(secerrors.ErrValidate, secerrors.IncompatibleSecretTypeErrorMsg, GCPSecretType, secret.Namespace, secret.Name)
	}
	if len(secret.Data) == 0 {
		return errors.Wrapf(secerrors.ErrValidate, secerrors.EmptySecretErrorMessage, secret.Namespace, secret.Name)
	}
	if _, ok := secret.Data[GCPProjectID]; !ok {
		return errors.Wrapf(secerrors.ErrValidate, secerrors.MissingRequiredFieldErrorMsg, GCPProjectID, secret.Namespace, secret.Name)
	}
	if _, ok := secret.Data[GCPServiceAccountJSONKey]; !ok {
		return errors.Wrapf(secerrors.ErrValidate, secerrors.MissingRequiredFieldErrorMsg, GCPServiceAccountJSONKey, secret.Namespace, secret.Name)
	}
	return nil
}
