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
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/kopia/command"
	"github.com/kanisterio/kanister/pkg/kopia/repository"
	reposerver "github.com/kanisterio/kanister/pkg/secrets/repositoryserver"
	"github.com/kanisterio/kanister/pkg/utils"
)

func (h *RepoServerHandler) connectToKopiaRepository() error {
	contentCacheMB, metadataCacheMB := h.getRepositoryCacheSettings()

	repoConfiguration := h.getRepositoryConfiguration()
	args := command.RepositoryCommandArgs{
		CommandArgs: &command.CommandArgs{
			RepoPassword:   string(h.RepositoryServerSecrets.repositoryPassword.Data[reposerver.RepoPasswordKey]),
			ConfigFilePath: repoConfiguration.ConfigFilePath,
			LogDirectory:   repoConfiguration.LogDirectory,
		},
		CacheDirectory:  repoConfiguration.CacheDirectory,
		Hostname:        h.RepositoryServer.Spec.Repository.Hostname,
		ContentCacheMB:  contentCacheMB,
		MetadataCacheMB: metadataCacheMB,
		Username:        h.RepositoryServer.Spec.Repository.Username,
		// TODO(Amruta): Generate path for respository
		RepoPathPrefix: h.RepositoryServer.Spec.Repository.RootPath,
		Location:       h.RepositoryServerSecrets.storage.Data,
	}

	return repository.ConnectToKopiaRepository(
		h.KubeCli,
		h.RepositoryServer.Namespace,
		h.RepositoryServer.Status.ServerInfo.PodName,
		repoServerPodContainerName,
		args,
	)
}

func (h *RepoServerHandler) getRepositoryConfiguration() v1alpha1.Configuration {
	configuration := v1alpha1.Configuration{
		ConfigFilePath: command.DefaultConfigFilePath,
		LogDirectory:   command.DefaultLogDirectory,
		CacheDirectory: command.DefaultCacheDirectory,
	}

	if h.RepositoryServer.Spec.Repository.Configuration.ConfigFilePath != "" {
		configuration.ConfigFilePath = h.RepositoryServer.Spec.Repository.Configuration.ConfigFilePath
	}
	if h.RepositoryServer.Spec.Repository.Configuration.LogDirectory != "" {
		configuration.LogDirectory = h.RepositoryServer.Spec.Repository.Configuration.LogDirectory
	}
	if h.RepositoryServer.Spec.Repository.Configuration.CacheDirectory != "" {
		configuration.CacheDirectory = h.RepositoryServer.Spec.Repository.Configuration.CacheDirectory
	}
	return configuration
}

func (h *RepoServerHandler) getRepositoryCacheSettings() (int, int) {
	defaultContentCacheMB, defaultMetadataCacheMB := command.GetGeneralCacheSizeSettings()
	contentCacheMB := defaultContentCacheMB
	metadataCacheMB := defaultMetadataCacheMB
	var err error
	if h.RepositoryServer.Spec.Repository.CacheSizeSettings.Content != "" {
		contentCacheMB, err = utils.GetIntOrDefault(h.RepositoryServer.Spec.Repository.CacheSizeSettings.Content, defaultContentCacheMB)
		if err != nil {
			h.Logger.Error(err, "cache content size should be an integer, using default value", field.M{"contentSize": h.RepositoryServer.Spec.Repository.CacheSizeSettings.Content, "default_value": defaultContentCacheMB})
		}
	}
	if h.RepositoryServer.Spec.Repository.CacheSizeSettings.Metadata != "" {
		metadataCacheMB, err = utils.GetIntOrDefault(h.RepositoryServer.Spec.Repository.CacheSizeSettings.Metadata, defaultMetadataCacheMB)
		if err != nil {
			h.Logger.Error(err, "cache metadata size should be an integer, using default value", field.M{"metadataSize": h.RepositoryServer.Spec.Repository.CacheSizeSettings.Metadata, "default_value": defaultMetadataCacheMB})
		}
	}

	return contentCacheMB, metadataCacheMB
}
