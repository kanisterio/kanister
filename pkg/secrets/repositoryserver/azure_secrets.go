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
	"github.com/kanisterio/errkit"
	corev1 "k8s.io/api/core/v1"

	secerrors "github.com/kanisterio/kanister/pkg/secrets/errors"
)

var _ Secret = &Azure{}

type Azure struct {
	storageLocation *corev1.Secret
}

func NewAzureLocation(secret *corev1.Secret) *Azure {
	return &Azure{
		storageLocation: secret,
	}
}

func (l *Azure) Validate() error {
	if l.storageLocation == nil {
		return errkit.Wrap(secerrors.ErrValidate, secerrors.NilSecretErrorMessage)
	}
	if len(l.storageLocation.Data) == 0 {
		return errkit.Wrap(secerrors.ErrValidate, secerrors.EmptySecretErrorMessage, l.storageLocation.Namespace, l.storageLocation.Name)
	}
	if _, ok := l.storageLocation.Data[BucketKey]; !ok {
		return errkit.Wrap(secerrors.ErrValidate, secerrors.MissingRequiredFieldErrorMsg, BucketKey, l.storageLocation.Namespace, l.storageLocation.Name)
	}
	return nil
}
