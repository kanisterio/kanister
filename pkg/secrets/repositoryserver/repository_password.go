package repositoryserver

import (
	"errors"

	v1 "k8s.io/api/core/v1"
)

const (
	RepoPasswordKey = "repo-password"
)

type RepositoryPassword struct {
	password *v1.Secret
}

func NewRepoPassword(secret *v1.Secret) *RepositoryPassword {
	return &RepositoryPassword{
		password: secret,
	}
}

func (r *RepositoryPassword) ValidateSecret() error {
	var count int
	if _, ok := r.password.Data[RepoPasswordKey]; !ok {
		return errors.New("repository password is required")
	}
	count++

	if len(r.password.Data) > count {
		return errors.New("repository password secret has an unknown field")
	}
	return nil
}
