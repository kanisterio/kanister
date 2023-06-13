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
)

var _ RepositoryServerSecret = &repositoryServerAdminCredentials{}

type repositoryServerAdminCredentials struct {
	credentials *corev1.Secret
}

func NewRepositoryServerAdminCredentials(secret *corev1.Secret) *repositoryServerAdminCredentials {
	return &repositoryServerAdminCredentials{
		credentials: secret,
	}
}

func (r *repositoryServerAdminCredentials) Validate() error {
	var count int
	if _, ok := r.credentials.Data[AdminUsernameKey]; !ok {
		return errors.Wrapf(ErrValidate, MissingRequiredFieldErrorMsg, AdminUsernameKey, r.credentials.Namespace, r.credentials.Name)
	}
	count++

	if _, ok := r.credentials.Data[AdminPasswordKey]; !ok {
		return errors.Wrapf(ErrValidate, MissingRequiredFieldErrorMsg, AdminPasswordKey, r.credentials.Namespace, r.credentials.Name)
	}
	count++

	if len(r.credentials.Data) > count {
		return errors.Wrapf(ErrValidate, UnknownFieldErrorMsg, r.credentials.Namespace, r.credentials.Name)
	}
	return nil
}
