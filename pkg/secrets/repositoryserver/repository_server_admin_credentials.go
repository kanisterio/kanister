package repositoryserver

import (
	"github.com/pkg/errors"

	v1 "k8s.io/api/core/v1"
)

const (
	RepositoryServerAdminUsernameKey = "username"
	RepositoryServerAdminPasswordKey = "password"
)

type RepositoryServerAdminCredentials struct {
	credentials *v1.Secret
}

func NewRepositoryServerAdminCredentials(secret *v1.Secret) *RepositoryServerAdminCredentials {
	return &RepositoryServerAdminCredentials{
		credentials: secret,
	}
}

func (r *RepositoryServerAdminCredentials) ValidateSecret() error {
	var count int
	if _, ok := r.credentials.Data[RepositoryServerAdminUsernameKey]; !ok {
		return errors.Wrapf(errValidate, "%s field is required in the kopia repository server admin credentials secret %s", RepositoryServerAdminUsernameKey, r.credentials.Name)
	}
	count++

	if _, ok := r.credentials.Data[RepositoryServerAdminPasswordKey]; !ok {
		return errors.Wrapf(errValidate, "%s field is required in the kopia repository server admin credentials secret %s", RepositoryServerAdminPasswordKey, r.credentials.Name)
	}
	count++

	if len(r.credentials.Data) > count {
		return errors.Wrapf(errValidate, "kopia repository server admin credentials secret %s has an unknown field", r.credentials.Name)
	}
	return nil
}
