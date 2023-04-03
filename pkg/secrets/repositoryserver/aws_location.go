package repositoryserver

import (
	"github.com/pkg/errors"

	v1 "k8s.io/api/core/v1"
)

const (
	AWSBucketKey string = "bucket"
	AWSRegionKey string = "region"
)

type AWS struct {
	storageLocation *v1.Secret
}

func NewAWSLocation(secret *v1.Secret) *AWS {
	return &AWS{
		storageLocation: secret,
	}
}

func (l *AWS) ValidateSecret() (err error) {
	if _, ok := l.storageLocation.Data[AWSBucketKey]; !ok {
		return errors.Wrapf(errValidate, "%s field is required in the kopia repository storage location secret %s", AWSBucketKey, l.storageLocation.Name)
	}
	if _, ok := l.storageLocation.Data[AWSRegionKey]; !ok {
		return errors.Wrapf(errValidate, "%s field is required in the kopia repository storage location secret %s", AWSRegionKey, l.storageLocation.Name)

	}

	return nil
}
