package function

import (
	"github.com/pkg/errors"

	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/secrets"
)

func ValidateCredentials(creds *param.Credential) error {
	if creds == nil {
		return errors.New("Empty credentials")
	}
	switch creds.Type {
	case param.CredentialTypeKeyPair:
		if creds.KeyPair == nil {
			return errors.New("Empty KeyPair field")
		}
		if len(creds.KeyPair.ID) == 0 {
			return errors.New("Access key ID is not set")
		}
		if len(creds.KeyPair.Secret) == 0 {
			return errors.New("Secret access key is not set")
		}
		return nil
	case param.CredentialTypeSecret:
		return secrets.ValidateCredentials(creds.Secret)
	default:
		return errors.Errorf("Unsupported type '%s' for credentials", creds.Type)
	}
}
