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

var _ Secret = &RepositoryServerAdminCredentials{}

type RepositoryServerAdminCredentials struct {
	credentials *corev1.Secret
}

func NewRepositoryServerAdminCredentials(secret *corev1.Secret) *RepositoryServerAdminCredentials {
	return &RepositoryServerAdminCredentials{
		credentials: secret,
	}
}

func (r *RepositoryServerAdminCredentials) Validate() error {
	if r.credentials == nil {
		return errkit.Wrap(secerrors.ErrValidate, secerrors.NilSecretErrorMessage)
	}
	if len(r.credentials.Data) == 0 {
		return errkit.Wrap(secerrors.ErrValidate, secerrors.EmptySecretErrorMessage, r.credentials.Namespace, r.credentials.Name)
	}

	// kopia repository server admin credentials secret must have exactly 2 fields
	if len(r.credentials.Data) != 2 {
		return errkit.Wrap(secerrors.ErrValidate, secerrors.UnknownFieldErrorMsg, r.credentials.Namespace, r.credentials.Name)
	}
	if _, ok := r.credentials.Data[AdminUsernameKey]; !ok {
		return errkit.Wrap(secerrors.ErrValidate, secerrors.MissingRequiredFieldErrorMsg, AdminUsernameKey, r.credentials.Namespace, r.credentials.Name)
	}
	if _, ok := r.credentials.Data[AdminPasswordKey]; !ok {
		return errkit.Wrap(secerrors.ErrValidate, secerrors.MissingRequiredFieldErrorMsg, AdminPasswordKey, r.credentials.Namespace, r.credentials.Name)
	}
	return nil
}
