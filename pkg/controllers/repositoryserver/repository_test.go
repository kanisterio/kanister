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

	//Test if Default cache size settings are set
	contentCacheMB, metadataCacheMB, cacheDirectory, err := repoServerHandler.getRepositoryCacheSettings()
	c.Assert(err, IsNil)
	c.Assert(contentCacheMB, Equals, defaultcontentCacheMB)
	c.Assert(metadataCacheMB, Equals, defaultmetadataCacheMB)
	c.Assert(cacheDirectory, Equals, command.DefaultCacheDirectory)

	//Test if configfured cache size settings are set
	repositoryServer.Spec.Repository.CacheSizeSettings = v1alpha1.CacheSizeSettings{
		Metadata:       "1000",
		Content:        "1100",
		CacheDirectory: "/tmp/test-cache",
	}
	contentCacheMB, metadataCacheMB, cacheDirectory, err = repoServerHandler.getRepositoryCacheSettings()
	c.Assert(err, IsNil)
	c.Assert(contentCacheMB, Equals, 1000)
	c.Assert(metadataCacheMB, Equals, 1100)
	c.Assert(metadataCacheMB, Equals, "/tmp/test-cache")

	// Check if default Content Cache size is set
	repositoryServer.Spec.Repository.CacheSizeSettings = v1alpha1.CacheSizeSettings{
		Metadata: "1000",
		Content:  "",
	}
	contentCacheMB, metadataCacheMB, _, err = repoServerHandler.getRepositoryCacheSettings()
	c.Assert(err, IsNil)
	c.Assert(contentCacheMB, Equals, defaultcontentCacheMB)
	c.Assert(metadataCacheMB, Equals, 1000)

	// Check if default Metadata Cache size is set
	repositoryServer.Spec.Repository.CacheSizeSettings = v1alpha1.CacheSizeSettings{
		Metadata: "",
		Content:  "1100",
	}
	contentCacheMB, metadataCacheMB, _, err = repoServerHandler.getRepositoryCacheSettings()
	c.Assert(err, IsNil)
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

	// Check if default Log directory and config file path is set
	configFilePath, logDirectory := repoServerHandler.getRepositoryCommonArguments()
	c.Assert(configFilePath, Equals, command.DefaultConfigFilePath)
	c.Assert(logDirectory, Equals, command.DefaultLogDirectory)

	// Check if custom Log directory and config file path is set
	repositoryServer.Spec.Repository.ConfigFilePath = "/tmp/test-config"
	repositoryServer.Spec.Repository.LogDirectory = "/tmp/test-log-directory"

	configFilePath, logDirectory = repoServerHandler.getRepositoryCommonArguments()
	c.Assert(configFilePath, Equals, "/tmp/test-config")
	c.Assert(logDirectory, Equals, "/tmp/test-log-directory")
}
