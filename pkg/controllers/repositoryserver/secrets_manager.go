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
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

type repositoryServerSecrets struct {
	storage            *corev1.Secret
	storageCredentials *corev1.Secret
	repositoryPassword *corev1.Secret
	serverAdmin        *corev1.Secret
	serverTLS          *corev1.Secret
	serverUserAccess   *corev1.Secret
}

// getSecretsFromCR fetches all the secrets in the RepositoryServer CR
func (h *RepoServerHandler) getSecretsFromCR(ctx context.Context) error {
	// TODO: For now, users should make sure all the secrets and the RepositoryServer CR are present in the
	//  same namespace. This namespace field can be overridden when we start creating secrets using 'kanctl' utility
	repositoryServer := h.RepositoryServer
	h.Logger.Info("Fetching secrets from all the secret references in the CR")
	storage, err := h.fetchSecret(ctx, &repositoryServer.Spec.Storage.SecretRef)
	if err != nil {
		return err
	}
	storageCredentials, err := h.fetchSecret(ctx, &repositoryServer.Spec.Storage.CredentialSecretRef)
	if err != nil {
		return err
	}
	repositoryPassword, err := h.fetchSecret(ctx, &repositoryServer.Spec.Repository.PasswordSecretRef)
	if err != nil {
		return err
	}
	serverAdmin, err := h.fetchSecret(ctx, &repositoryServer.Spec.Server.AdminSecretRef)
	if err != nil {
		return err
	}
	serverTLS, err := h.fetchSecret(ctx, &repositoryServer.Spec.Server.TLSSecretRef)
	if err != nil {
		return err
	}
	serverUserAccess, err := h.fetchSecret(ctx, &repositoryServer.Spec.Server.UserAccess.UserAccessSecretRef)
	if err != nil {
		return err
	}
	secrets := repositoryServerSecrets{
		storage:            storage,
		storageCredentials: storageCredentials,
		repositoryPassword: repositoryPassword,
		serverAdmin:        serverAdmin,
		serverTLS:          serverTLS,
		serverUserAccess:   serverUserAccess,
	}
	h.RepositoryServerSecrets = secrets
	return nil
}

func (h *RepoServerHandler) fetchSecret(ctx context.Context, ref *corev1.SecretReference) (*corev1.Secret, error) {
	if ref == nil {
		return nil, errors.New("repository server CR does not have a secret reference set")
	}

	h.Logger.Info(fmt.Sprintf("Fetching secret %s from namespace %s", ref.Name, ref.Namespace))
	secret := corev1.Secret{}
	err := h.Reconciler.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: ref.Namespace}, &secret)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Error fetching secret %s from namespace %s", ref.Name, ref.Namespace))
	}
	return &secret, nil
}
