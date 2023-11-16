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
	corev1 "k8s.io/api/core/v1"

	secerrors "github.com/kanisterio/kanister/pkg/secrets/errors"
)

type s3Compliant struct {
	storageLocation *corev1.Secret
}

func NewS3CompliantLocation(secret *corev1.Secret) *s3Compliant {
	return &s3Compliant{
		storageLocation: secret,
	}
}

var _ Secret = &s3Compliant{}

func (s s3Compliant) Validate() error {
	if s.storageLocation == nil {
		return errors.Wrapf(secerrors.ErrValidate, secerrors.NilSecretErrorMessage)
	}
	if len(s.storageLocation.Data) == 0 {
		return errors.Wrapf(secerrors.ErrValidate, secerrors.EmptySecretErrorMessage, s.storageLocation.Namespace, s.storageLocation.Name)
	}
	if _, ok := s.storageLocation.Data[BucketKey]; !ok {
		return errors.Wrapf(secerrors.ErrValidate, secerrors.MissingRequiredFieldErrorMsg, BucketKey, s.storageLocation.Namespace, s.storageLocation.Name)
	}
	if _, ok := s.storageLocation.Data[EndpointKey]; !ok {
		return errors.Wrapf(secerrors.ErrValidate, secerrors.MissingRequiredFieldErrorMsg, EndpointKey, s.storageLocation.Namespace, s.storageLocation.Name)
	}
	if _, ok := s.storageLocation.Data[RegionKey]; !ok {
		return errors.Wrapf(secerrors.ErrValidate, secerrors.MissingRequiredFieldErrorMsg, RegionKey, s.storageLocation.Namespace, s.storageLocation.Name)
	}
	return nil
}
