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
	"github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kopia/command"
	"github.com/kanisterio/kanister/pkg/testutil"
	. "gopkg.in/check.v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (s *RepoServerControllerSuite) TestCacheSizeConfiguration(c *C) {
	repositoryServer := testutil.GetTestKopiaRepositoryServerCR(s.repoServerControllerNamespace)
	setRepositoryServerSecretsInCR(&s.repoServerSecrets, &repositoryServer)

	defaultcontentCacheMB, defaultmetadataCacheMB := command.GetGeneralCacheSizeSettings()

	repoServerHandler := RepoServerHandler{
		Req:              reconcile.Request{},
		Reconciler:       s.DefaultRepoServerReconciler,
		KubeCli:          s.kubeCli,
		RepositoryServer: &repositoryServer,
	}

	// Test if Default cache size settings are set
	contentCacheMB, metadataCacheMB := repoServerHandler.getRepositoryCacheSettings()
	c.Assert(contentCacheMB, Equals, defaultcontentCacheMB)
	c.Assert(metadataCacheMB, Equals, defaultmetadataCacheMB)

	// Test if configfured cache size settings are set
	repositoryServer.Spec.Repository.CacheSizeSettings = v1alpha1.CacheSizeSettings{
		Metadata: "1000",
		Content:  "1100",
	}
	contentCacheMB, metadataCacheMB = repoServerHandler.getRepositoryCacheSettings()
	c.Assert(contentCacheMB, Equals, 1100)
	c.Assert(metadataCacheMB, Equals, 1000)

	// Check if default Content Cache size is set
	repositoryServer.Spec.Repository.CacheSizeSettings = v1alpha1.CacheSizeSettings{
		Metadata: "1000",
		Content:  "",
	}
	contentCacheMB, metadataCacheMB = repoServerHandler.getRepositoryCacheSettings()
	c.Assert(contentCacheMB, Equals, defaultcontentCacheMB)
	c.Assert(metadataCacheMB, Equals, 1000)

	// Check if default Metadata Cache size is set
	repositoryServer.Spec.Repository.CacheSizeSettings = v1alpha1.CacheSizeSettings{
		Metadata: "",
		Content:  "1100",
	}
	contentCacheMB, metadataCacheMB = repoServerHandler.getRepositoryCacheSettings()
	c.Assert(contentCacheMB, Equals, 1100)
	c.Assert(metadataCacheMB, Equals, defaultmetadataCacheMB)
}

func (s *RepoServerControllerSuite) TestConfigFileAndLogDirectoryConfiguration(c *C) {
	repositoryServer := testutil.GetTestKopiaRepositoryServerCR(s.repoServerControllerNamespace)
	setRepositoryServerSecretsInCR(&s.repoServerSecrets, &repositoryServer)

	repoServerHandler := RepoServerHandler{
		Req:              reconcile.Request{},
		Reconciler:       s.DefaultRepoServerReconciler,
		KubeCli:          s.kubeCli,
		RepositoryServer: &repositoryServer,
	}

	// Check if default values for log directory,config file path and cache directory are set
	configuration := repoServerHandler.getRepositoryConfiguration()
	c.Assert(configuration.ConfigFilePath, Equals, command.DefaultConfigFilePath)
	c.Assert(configuration.LogDirectory, Equals, command.DefaultLogDirectory)
	c.Assert(configuration.CacheDirectory, Equals, command.DefaultCacheDirectory)

	// Check if custom values for log directory,config file path and cache directory are set
	repositoryServer.Spec.Repository.Configuration.ConfigFilePath = "/tmp/test-config"
	repositoryServer.Spec.Repository.Configuration.LogDirectory = "/tmp/test-log-directory"
	repositoryServer.Spec.Repository.Configuration.CacheDirectory = "/tmp/test-cache-directory"

	configuration = repoServerHandler.getRepositoryConfiguration()
	c.Assert(configuration.ConfigFilePath, Equals, "/tmp/test-config")
	c.Assert(configuration.LogDirectory, Equals, "/tmp/test-log-directory")
	c.Assert(configuration.CacheDirectory, Equals, "/tmp/test-cache-directory")
}
