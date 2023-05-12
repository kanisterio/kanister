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

func ValidateGCPCredentials(secret *v1.Secret) error {
	if string(secret.Type) != GCPSecretType {
		return errors.New("Secret is not GCP secret")
	}
	if _, ok := secret.Data[GCPProjectID]; !ok {
		return errors.Wrapf(repositoryserver.ErrValidate, "%s field is required in the kopia repository storage location secret %s", GCPProjectID, secret.Name)
	}
	if _, ok := secret.Data[GCPServiceAccountJsonKey]; !ok {
		return errors.Wrapf(repositoryserver.ErrValidate, "%s field is required in the kopia repository storage location secret %s", GCPServiceAccountJsonKey, secret.Name)
	}
	return nil
}
