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

type RepositoryServerUserAccessCredentials struct {
	credentials *corev1.Secret
}

var _ Secret = &RepositoryServerUserAccessCredentials{}

func NewRepositoryServerUserAccessCredentials(secret *corev1.Secret) *RepositoryServerUserAccessCredentials {
	return &RepositoryServerUserAccessCredentials{
		credentials: secret,
	}
}

func (r *RepositoryServerUserAccessCredentials) Validate() error {
	if r.credentials == nil {
		return errkit.Wrap(secerrors.ErrValidate, secerrors.NilSecretErrorMessage)
	}
	if len(r.credentials.Data) == 0 {
		return errkit.Wrap(secerrors.ErrValidate, secerrors.EmptySecretErrorMessage, r.credentials.Namespace, r.credentials.Name)
	}
	return nil
}
