package repositoryserver

import (
	v1 "k8s.io/api/core/v1"
)

type RepositoryServerUserAccessCredentials struct {
	credentials *v1.Secret
}

func NewRepositoryServerUserAccessCredentials(secret *v1.Secret) *RepositoryServerUserAccessCredentials {
	return &RepositoryServerUserAccessCredentials{
		credentials: secret,
	}
}

func (r *RepositoryServerUserAccessCredentials) ValidateSecret() error {
	return nil
}
