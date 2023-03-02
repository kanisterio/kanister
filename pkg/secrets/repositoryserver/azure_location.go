package repositoryserver

import v1 "k8s.io/api/core/v1"

type Azure struct {
	storageLocation *v1.Secret
}

func NewAzureLocation(secret *v1.Secret) *Azure {
	return &Azure{
		storageLocation: secret,
	}
}

func (l *Azure) ValidateSecret() error {
	return nil
}
