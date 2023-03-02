package secrets

import (
	"github.com/kanisterio/kanister/pkg/secrets/repositoryserver"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
)

const (
	// AWSLocationSecretType represents the storage location secret type for AWS
	AWSLocationSecretType string = "secrets.kanister.io/aws-location"
	// AWSLocationSecretType represents the storage location secret type for Azure
	AzureLocationSecretType string = "secrets.kanister.io/azure-location"
	// GCPLocationSecretType represents the storage location secret type for AWS
	GCPLocationSecretType string = "secrets.kanister.io/gcp-location"
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

func ValidateRepositoryServerSecret(repositoryServerSecret *v1.Secret) error {
	var secret repositoryserver.RepositoryServerSecrets
	switch string(repositoryServerSecret.Type) {
	case AWSLocationSecretType:
		secret = repositoryserver.NewAWSLocation(repositoryServerSecret)
	case AzureLocationSecretType:
		secret = repositoryserver.NewAzureLocation(repositoryServerSecret)
	case RepositoryPasswordSecretType:
		secret = repositoryserver.NewRepoPassword(repositoryServerSecret)
	case RepositoryServerAdminCredentialsSecretType:
		secret = repositoryserver.NewRepositoryServerAdminCredentials(repositoryServerSecret)
	default:
		return ValidateCredentials(repositoryServerSecret)
	}
	return secret.ValidateSecret()
}
