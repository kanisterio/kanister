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
	"strconv"

	"github.com/kanisterio/kanister/pkg/kopia/command"
	"github.com/kanisterio/kanister/pkg/kopia/repository"
	reposerver "github.com/kanisterio/kanister/pkg/secrets/repositoryserver"
)

func (h *RepoServerHandler) connectToKopiaRepository() error {
	contentCacheMB, metadataCacheMB, cacheDirectory, err := h.getRepositoryCacheSettings()
	if err != nil {
		return err
	}
	configFilePath, logDirectory := h.getRepositoryCommonArguments()
	args := command.RepositoryCommandArgs{
		CommandArgs: &command.CommandArgs{
			RepoPassword:   string(h.RepositoryServerSecrets.repositoryPassword.Data[reposerver.RepoPasswordKey]),
			ConfigFilePath: configFilePath,
			LogDirectory:   logDirectory,
		},
		CacheDirectory:  cacheDirectory,
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

func (h *RepoServerHandler) getRepositoryCommonArguments() (configFilePath, logDirectory string) {
	configFilePath = command.DefaultConfigFilePath
	logDirectory = command.DefaultLogDirectory
	if h.RepositoryServer.Spec.Repository.ConfigFilePath != "" {
		configFilePath = h.RepositoryServer.Spec.Repository.ConfigFilePath
	}
	if h.RepositoryServer.Spec.Repository.LogDirectory != "" {
		logDirectory = h.RepositoryServer.Spec.Repository.LogDirectory
	}
	return
}

func (h *RepoServerHandler) getRepositoryCacheSettings() (contentCacheMB, metadataCacheMB int, cacheDirectory string, err error) {
	contentCacheMB, metadataCacheMB = command.GetGeneralCacheSizeSettings()
	cacheDirectory = command.DefaultCacheDirectory
	if h.RepositoryServer.Spec.Repository.CacheSizeSettings.Content != "" {
		contentCacheMB, err = strconv.Atoi(h.RepositoryServer.Spec.Repository.CacheSizeSettings.Content)
		if err != nil {
			return
		}
	}
	if h.RepositoryServer.Spec.Repository.CacheSizeSettings.Metadata != "" {
		metadataCacheMB, err = strconv.Atoi(h.RepositoryServer.Spec.Repository.CacheSizeSettings.Metadata)
		if err != nil {
			return
		}
	}
	if h.RepositoryServer.Spec.Repository.CacheSizeSettings.CacheDirectory != "" {
		cacheDirectory = h.RepositoryServer.Spec.Repository.CacheSizeSettings.CacheDirectory
	}
	return
}
