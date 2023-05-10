// Copyright 2023 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
