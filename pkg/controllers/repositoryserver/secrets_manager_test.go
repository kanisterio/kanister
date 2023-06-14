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

	. "gopkg.in/check.v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (s *RepoServerControllerSuite) TestSuccessfulFetchSecretsForRepositoryServer(c *C) {
	// Test getSecretsFromCR is successfull
	repositoryServer := getDefaultKopiaRepositoryServerCR(s.repoServerControllerNamespace)
	setRepositoryServerSecretsInCR(&s.repoServerSecrets, repositoryServer)
	repoServerHandler := RepoServerHandler{
		Req:              reconcile.Request{},
		Reconciler:       s.DefaultRepoServerReconciler,
		KubeCli:          s.kubeCli,
		RepositoryServer: repositoryServer,
	}
	err := repoServerHandler.getSecretsFromCR(context.Background())
	c.Assert(err, IsNil)
	c.Assert(repoServerHandler.RepositoryServerSecrets, NotNil)
	c.Assert(repoServerHandler.RepositoryServerSecrets.repositoryPassword, NotNil)
	c.Assert(repoServerHandler.RepositoryServerSecrets.serverAdmin, NotNil)
	c.Assert(repoServerHandler.RepositoryServerSecrets.serverUserAccess, NotNil)
	c.Assert(repoServerHandler.RepositoryServerSecrets.serverTLS, NotNil)
	c.Assert(repoServerHandler.RepositoryServerSecrets.storage, NotNil)
	c.Assert(repoServerHandler.RepositoryServerSecrets.storageCredentials, NotNil)
	c.Assert(repoServerHandler.RepositoryServerSecrets.serverAdmin.Name, Equals, s.repoServerSecrets.serverAdmin.Name)
	c.Assert(repoServerHandler.RepositoryServerSecrets.serverTLS.Name, Equals, s.repoServerSecrets.serverTLS.Name)
	c.Assert(repoServerHandler.RepositoryServerSecrets.repositoryPassword.Name, Equals, s.repoServerSecrets.repositoryPassword.Name)
	c.Assert(repoServerHandler.RepositoryServerSecrets.serverUserAccess.Name, Equals, s.repoServerSecrets.serverUserAccess.Name)
	c.Assert(repoServerHandler.RepositoryServerSecrets.storageCredentials.Name, Equals, s.repoServerSecrets.storageCredentials.Name)
	c.Assert(repoServerHandler.RepositoryServerSecrets.storage.Name, Equals, s.repoServerSecrets.storage.Name)

	c.Assert(repoServerHandler.RepositoryServerSecrets.serverAdmin.Namespace, Equals, s.repoServerSecrets.serverAdmin.Namespace)
	c.Assert(repoServerHandler.RepositoryServerSecrets.serverTLS.Namespace, Equals, s.repoServerSecrets.serverTLS.Namespace)
	c.Assert(repoServerHandler.RepositoryServerSecrets.repositoryPassword.Namespace, Equals, s.repoServerSecrets.repositoryPassword.Namespace)
	c.Assert(repoServerHandler.RepositoryServerSecrets.serverUserAccess.Namespace, Equals, s.repoServerSecrets.serverUserAccess.Namespace)
	c.Assert(repoServerHandler.RepositoryServerSecrets.storageCredentials.Namespace, Equals, s.repoServerSecrets.storageCredentials.Namespace)
	c.Assert(repoServerHandler.RepositoryServerSecrets.storage.Namespace, Equals, s.repoServerSecrets.storage.Namespace)
}

func (s *RepoServerControllerSuite) TestUnsuccessfulFetchSecretsForRepositoryServer(c *C) {
	// Test getSecretsFromCR is unsuccesful when one of the secrets does not exist in the namespace
	repositoryServer := getDefaultKopiaRepositoryServerCR(s.repoServerControllerNamespace)
	setRepositoryServerSecretsInCR(&s.repoServerSecrets, repositoryServer)
	repositoryServer.Spec.Storage.SecretRef.Name = "SecretDoesNotExist"
	repoServerHandler := RepoServerHandler{RepositoryServer: repositoryServer}
	repoServerHandler.RepositoryServer = repositoryServer
	err := repoServerHandler.getSecretsFromCR(context.Background())
	c.Assert(err, NotNil)
	c.Assert(repoServerHandler.RepositoryServerSecrets, IsNil)
}
