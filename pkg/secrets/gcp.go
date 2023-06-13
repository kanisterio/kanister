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
	v1 "k8s.io/api/core/v1"

	"github.com/kanisterio/kanister/pkg/secrets/repositoryserver"
)

const (
	// GCPProjectID is the config map key for gcp project id data
	GCPProjectID string = "gcp_project_id"
	// GCPServiceKey is the config map key for gcp service key data
	GCPServiceKey string = "gcp_service_key"
	// GCPServerAccountJsonKey is the key for gcp service account json
	GCPServiceAccountJsonKey string = "service-account.json"

	GCPSecretType string = "secrets.kanister.io/gcp"
)

// ValidateGCPCredentials function is to verify the schema of GCP secrets
// that need to be provided for kopia commands
// Required fields:
// - GCPProjectID
// - GCPServiceAccountJsonKey
func ValidateGCPCredentials(secret *v1.Secret) error {
	if string(secret.Type) != GCPSecretType {
		return errors.Errorf("The type of the secret is incorrect,it is not a GCP compatible secret, the type of the secret should be %s", GCPSecretType)
	}
	if _, ok := secret.Data[GCPProjectID]; !ok {
		return errors.Wrapf(repositoryserver.ErrValidate, "%s field is required in the kopia repository storage credentials secret %s", GCPProjectID, secret.Name)
	}
	if _, ok := secret.Data[GCPServiceAccountJsonKey]; !ok {
		return errors.Wrapf(repositoryserver.ErrValidate, "%s field is required in the kopia repository storage credentials secret %s", GCPServiceAccountJsonKey, secret.Name)
	}
	return nil
}
