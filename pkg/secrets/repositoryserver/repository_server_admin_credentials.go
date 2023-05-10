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
	RepositoryServerAdminUsernameKey = "username"
	RepositoryServerAdminPasswordKey = "password"
)

type RepositoryServerAdminCredentials struct {
	credentials *v1.Secret
}

func NewRepositoryServerAdminCredentials(secret *v1.Secret) *RepositoryServerAdminCredentials {
	return &RepositoryServerAdminCredentials{
		credentials: secret,
	}
}

func (r *RepositoryServerAdminCredentials) Validate() error {
	var count int
	if _, ok := r.credentials.Data[RepositoryServerAdminUsernameKey]; !ok {
		return errors.Wrapf(errValidate, "%s field is required in the kopia repository server admin credentials secret %s", RepositoryServerAdminUsernameKey, r.credentials.Name)
	}
	count++

	if _, ok := r.credentials.Data[RepositoryServerAdminPasswordKey]; !ok {
		return errors.Wrapf(errValidate, "%s field is required in the kopia repository server admin credentials secret %s", RepositoryServerAdminPasswordKey, r.credentials.Name)
	}
	count++

	if len(r.credentials.Data) > count {
		return errors.Wrapf(errValidate, "kopia repository server admin credentials secret %s has an unknown field", r.credentials.Name)
	}
	return nil
}
