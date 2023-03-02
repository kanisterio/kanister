package repositoryserver

import (
	"errors"

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
		return errors.New("repository server admin username is required")
	}
	count++

	if _, ok := r.credentials.Data[RepositoryServerAdminPasswordKey]; !ok {
		return errors.New("repository server admin password is required")
	}
	count++

	if len(r.credentials.Data) > count {
		return errors.New("repository server admin credentials secret has an unknown field")
	}
	return nil
}
