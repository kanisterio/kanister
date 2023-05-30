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

package repositoryserver

import (
	"context"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kanisterio/kanister/pkg/client/clientset/versioned"
	"github.com/kanisterio/kanister/pkg/kube"
)

// Do validate the RepositoryServer object and make sure that
// update is not allowed for the following fields:
// - .spec.storage.secretRef
// - .spec.repository.rootPath
// - .spec.server.adminSecretRef
// - .spec.server.tlsSecretRef
func Do(updatedObject *crv1alpha1.RepositoryServer) error {
	config, err := kube.LoadConfig()
	if err != nil {
		return err
	}
	crCli, err := versioned.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "could not get the CRD client")
	}
	ctx := context.Background()
	originalObject, err := crCli.CrV1alpha1().RepositoryServers(updatedObject.Namespace).Get(ctx, updatedObject.Name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "could not get the RepositoryServer %s/%s", updatedObject.Namespace, updatedObject.Name)
	}
	return validateRepositoryServerParamsUpdate(originalObject, updatedObject)
}

func validateRepositoryServerParamsUpdate(originalObject, updatedObject *crv1alpha1.RepositoryServer) error {
	if originalObject.Spec.Storage.SecretRef != updatedObject.Spec.Storage.SecretRef {
		return errors.Errorf("cannot update the secretRef of the RepositoryServer %s/%s", originalObject.Namespace, originalObject.Name)
	}
	if originalObject.Spec.Repository.RootPath != updatedObject.Spec.Repository.RootPath {
		return errors.Errorf("cannot update the rootPath of the RepositoryServer %s/%s", originalObject.Namespace, originalObject.Name)
	}
	if originalObject.Spec.Server.AdminSecretRef != updatedObject.Spec.Server.AdminSecretRef {
		return errors.Errorf("cannot update the adminSecretRef of the RepositoryServer %s/%s", originalObject.Namespace, originalObject.Name)
	}
	if originalObject.Spec.Server.TLSSecretRef != updatedObject.Spec.Server.TLSSecretRef {
		return errors.Errorf("cannot update the tlsSecretRef of the RepositoryServer %s/%s", originalObject.Namespace, originalObject.Name)
	}
	return nil
}
