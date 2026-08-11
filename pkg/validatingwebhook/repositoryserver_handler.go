// Copyright 2023 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package validatingwebhook

import (
	"context"
	"fmt"

	"github.com/kanisterio/errkit"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
)

type RepositoryServerValidator struct{}

var _ admission.Validator[*crv1alpha1.RepositoryServer] = &RepositoryServerValidator{}

//nolint:lll
//+kubebuilder:webhook:path=/validate/v1alpha1/repositoryserver,mutating=false,failurePolicy=fail,sideEffects=None,groups=cr.kanister.io,resources=repositoryservers,verbs=update,versions=v1alpha1,name=repositoryserver.cr.kanister.io,admissionReviewVersions=v1

// ValidateCreate implements admission.Validator so a webhook will be registered for the type
func (r *RepositoryServerValidator) ValidateCreate(ctx context.Context, obj *crv1alpha1.RepositoryServer) (admission.Warnings, error) {
	return nil, nil
}

// ValidateUpdate implements admission.Validator so a webhook will be registered for the type
func (r *RepositoryServerValidator) ValidateUpdate(ctx context.Context, old *crv1alpha1.RepositoryServer, new *crv1alpha1.RepositoryServer) (admission.Warnings, error) {
	if old == nil || new == nil {
		return nil, errkit.New("Either updated object or the old object is not of type RepositoryServer.cr.kanister.io")
	}
	errMsg := fmt.Sprintf("RepositoryServer.cr.kanister.io \"%s\" is invalid: spec.repository.rootPath: Invalid value, Value is immutable", new.Name)
	if old.Spec.Repository.RootPath != new.Spec.Repository.RootPath {
		return nil, errkit.New(errMsg)
	}
	return nil, nil
}

// ValidateDelete implements admission.Validator so a webhook will be registered for the type
func (r *RepositoryServerValidator) ValidateDelete(ctx context.Context, obj *crv1alpha1.RepositoryServer) (admission.Warnings, error) {
	return nil, nil
}
