package repositoryserver

import v1 "k8s.io/api/core/v1"

type GCP struct {
	storageLocation *v1.Secret
}

func NewGCPLocation(secret *v1.Secret) *GCP {
	return &GCP{
		storageLocation: secret,
	}
}

func (l *GCP) ValidateSecret() error {
	return nil
}
