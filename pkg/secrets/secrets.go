package secrets

import (
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
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
	default:
		return errors.Errorf("Unsupported type '%s' for secret '%s:%s'", string(secret.Type), secret.Namespace, secret.Name)
	}
}
