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

var _ Secret = &repositoryPassword{}

type repositoryPassword struct {
	password *corev1.Secret
}

func NewRepoPassword(secret *corev1.Secret) *repositoryPassword {
	return &repositoryPassword{
		password: secret,
	}
}

// Validate the kopia repository password for required fields as well as unknown fields
func (r *repositoryPassword) Validate() error {
	if r.password == nil {
		return errors.Wrapf(secerrors.ErrValidate, secerrors.NilSecretErrorMessage)
	}
	if len(r.password.Data) == 0 {
		return errors.Wrapf(secerrors.ErrValidate, secerrors.EmptySecretErrorMessage, r.password.Namespace, r.password.Name)
	}
	// kopia repository must have exactly 1 field
	if len(r.password.Data) != 1 {
		return errors.Wrapf(secerrors.ErrValidate, secerrors.UnknownFieldErrorMsg, r.password.Namespace, r.password.Name)
	}
	if _, ok := r.password.Data[RepoPasswordKey]; !ok {
		return errors.Wrapf(secerrors.ErrValidate, secerrors.MissingRequiredFieldErrorMsg, RepoPasswordKey, r.password.Namespace, r.password.Name)
	}
	return nil
}
