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

	"github.com/kanisterio/kanister/pkg/testutil"
)

func (s *RepoServerControllerSuite) TestFetchSecretsForRepositoryServer(c *C) {
	// Test getSecretsFromCR is successful
	repositoryServer := testutil.GetTestKopiaRepositoryServerCR(s.repoServerControllerNamespace)
	setRepositoryServerSecretsInCR(&s.repoServerSecrets, &repositoryServer)

	repoServerHandler := RepoServerHandler{
		Req:              reconcile.Request{},
		Reconciler:       s.DefaultRepoServerReconciler,
		KubeCli:          s.kubeCli,
		RepositoryServer: &repositoryServer,
	}

	err := repoServerHandler.getSecretsFromCR(context.Background())
	c.Assert(err, IsNil)
	c.Assert(repoServerHandler.RepositoryServerSecrets, NotNil)
	c.Assert(repoServerHandler.RepositoryServerSecrets.repositoryPassword, DeepEquals, s.repoServerSecrets.repositoryPassword)
	c.Assert(repoServerHandler.RepositoryServerSecrets.storage, DeepEquals, s.repoServerSecrets.storage)
	c.Assert(repoServerHandler.RepositoryServerSecrets.storageCredentials, DeepEquals, s.repoServerSecrets.storageCredentials)
	c.Assert(repoServerHandler.RepositoryServerSecrets.serverAdmin, DeepEquals, s.repoServerSecrets.serverAdmin)
	c.Assert(repoServerHandler.RepositoryServerSecrets.serverTLS, DeepEquals, s.repoServerSecrets.serverTLS)
	c.Assert(repoServerHandler.RepositoryServerSecrets.serverUserAccess, DeepEquals, s.repoServerSecrets.serverUserAccess)

	// Test getSecretsFromCR is unsuccessful when one of the secrets does not exist in the namespace
	repositoryServer.Spec.Storage.SecretRef.Name = "SecretDoesNotExist"
	repoServerHandler.RepositoryServerSecrets = repositoryServerSecrets{}
	err = repoServerHandler.getSecretsFromCR(context.Background())
	c.Assert(err, NotNil)
	c.Assert(repoServerHandler.RepositoryServerSecrets, Equals, repositoryServerSecrets{})
}
