// Copyright 2022 The Kanister Authors.
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

	"github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

const errValidate = "Validation Error"

type RepositoryServerWebhook struct {
}

var _ webhook.CustomValidator = &RepositoryServerWebhook{}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate/v1alpha1/repositoryserver,mutating=false,failurePolicy=fail,sideEffects=None,groups=cr.kanister.io,resources=repositoryservers,verbs=update,versions=v1alpha1,name=repositoryserver.cr.kanister.io,admissionReviewVersions=v1

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *RepositoryServerWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *RepositoryServerWebhook) ValidateUpdate(ctx context.Context, old runtime.Object, new runtime.Object) error {
	oldrs, ook := old.(*v1alpha1.RepositoryServer)
	newrs, nok := old.(*v1alpha1.RepositoryServer)
	if !ook || !nok {
		return errors.New("RepositoryServer.cr.kanister.io object expected")
	}
	errMsg := fmt.Sprintf("RepositoryServer.cr.kanister.io \"%s\" is invalid: spec.repository.rootPath: Invalid value, Value is immutable", newrs.Name)
	if oldrs.Spec.Repository.RootPath != newrs.Spec.Repository.RootPath {
		return errors.New(errMsg)
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *RepositoryServerWebhook) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	return nil
}
