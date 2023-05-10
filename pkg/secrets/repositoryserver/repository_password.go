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
	"fmt"

	"github.com/pkg/errors"

	v1 "k8s.io/api/core/v1"
)

const (
	RepoPasswordKey = "repo-password"
)

var errValidate = fmt.Errorf("validation Failed")

type RepositoryPassword struct {
	password *v1.Secret
}

func NewRepoPassword(secret *v1.Secret) *RepositoryPassword {
	return &RepositoryPassword{
		password: secret,
	}
}

// ValidateSecret validates the kopia repository password for required fields as well as unknown fields
func (r *RepositoryPassword) ValidateSecret() error {
	var count int
	if _, ok := r.password.Data[RepoPasswordKey]; !ok {
		return errors.Wrapf(errValidate, "%s field is required in the kopia repository password secret %s", RepoPasswordKey, r.password.Name)
	}
	count++

	if len(r.password.Data) > count {
		return errors.Wrapf(errValidate, "kopia repository password secret %s has an unknown field", r.password.Name)
	}
	return nil
}
