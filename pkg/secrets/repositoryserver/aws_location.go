package repositoryserver

import (
	"errors"

	v1 "k8s.io/api/core/v1"
)

const (
	AWSBucketKey   string = "bucket"
	AWSEndpointKey string = "endpoint"
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
		return errors.New("AWS Bucket is required")
	}
	if _, ok := l.storageLocation.Data[AWSEndpointKey]; !ok {
		return errors.New("AWS Endpoint is required")
	}

	return nil
}
