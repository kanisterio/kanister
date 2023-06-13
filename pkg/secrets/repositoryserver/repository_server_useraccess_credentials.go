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

var _ Secret = &repositoryServerUserAccessCredentials{}

type repositoryServerUserAccessCredentials struct {
	credentials *corev1.Secret
}

func NewRepositoryServerUserAccessCredentials(secret *corev1.Secret) *repositoryServerUserAccessCredentials {
	return &repositoryServerUserAccessCredentials{
		credentials: secret,
	}
}

func (r *repositoryServerUserAccessCredentials) Validate() error {
	if r.credentials == nil {
		return errors.Wrapf(ErrValidate, NilSecretErrorMessage)
	}
	if len(r.credentials.Data) == 0 {
		return errors.Wrapf(ErrValidate, EmptySecretErrorMessage, r.credentials.Namespace, r.credentials.Name)
	}
	return nil
}
