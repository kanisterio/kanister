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
	"github.com/kanisterio/kanister/pkg/kopia/command"
	"github.com/kanisterio/kanister/pkg/kopia/repository"
	reposerver "github.com/kanisterio/kanister/pkg/secrets/repositoryserver"
)

func (h *RepoServerHandler) connectToKopiaRepository(ctx context.Context) error {
	repoConfiguration := h.getRepositoryConfiguration()
	cacheSizeSettings := h.getRepositoryCacheSettings()
	args := command.RepositoryCommandArgs{
		CommandArgs: &command.CommandArgs{
			RepoPassword:   string(h.RepositoryServerSecrets.repositoryPassword.Data[reposerver.RepoPasswordKey]),
			ConfigFilePath: repoConfiguration.ConfigFilePath,
			LogDirectory:   repoConfiguration.LogDirectory,
		},
		CacheDirectory: repoConfiguration.CacheDirectory,
		Hostname:       h.RepositoryServer.Spec.Repository.Hostname,
		CacheArgs: command.CacheArgs{
			ContentCacheLimitMB:  *cacheSizeSettings.Content,
			MetadataCacheLimitMB: *cacheSizeSettings.Metadata,
		},
		Username: h.RepositoryServer.Spec.Repository.Username,
		// TODO(Amruta): Generate path for repository
		RepoPathPrefix: h.RepositoryServer.Spec.Repository.RootPath,
		Location:       h.RepositoryServerSecrets.storage.Data,
	}

	return repository.ConnectToKopiaRepository(
		ctx,
		h.KubeCli,
		h.RepositoryServer.Namespace,
		h.RepositoryServer.Status.ServerInfo.PodName,
		repoServerPodContainerName,
		args,
	)
}

func (h *RepoServerHandler) getRepositoryConfiguration() crv1alpha1.Configuration {
	configuration := crv1alpha1.Configuration{
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

func (h *RepoServerHandler) getRepositoryCacheSettings() crv1alpha1.CacheSizeSettings {
	defaultContentCacheMB, defaultMetadataCacheMB := command.GetGeneralCacheSizeSettings()
	cacheSizeSettings := crv1alpha1.CacheSizeSettings{
		Metadata: &defaultMetadataCacheMB,
		Content:  &defaultContentCacheMB,
	}
	if h.RepositoryServer.Spec.Repository.CacheSizeSettings.Content != nil {
		cacheSizeSettings.Content = h.RepositoryServer.Spec.Repository.CacheSizeSettings.Content
	}
	if h.RepositoryServer.Spec.Repository.CacheSizeSettings.Metadata != nil {
		cacheSizeSettings.Metadata = h.RepositoryServer.Spec.Repository.CacheSizeSettings.Metadata
	}
	return cacheSizeSettings
}
