package repositoryserver

import (
	"strconv"

	"github.com/kanisterio/kanister/pkg/kopia/command"
	"github.com/kanisterio/kanister/pkg/kopia/repository"
)

const (
	repoPasswordKey           = "repo-password"
	defaultRepoConfigFilePath = "/tmp/config.file"
	defaultRepoLogDirectory   = "/tmp/log.dir"
	defaultCacheDirectory     = "/tmp/cache.dir"
)

func (h *RepoServerHandler) connectToKopiaRepository() error {
	contentCacheMB, metadataCacheMB, err := h.getRepositoryCacheSettings()
	if err != nil {
		return err
	}
	args := command.RepositoryCommandArgs{
		CommandArgs: &command.CommandArgs{
			RepoPassword:   string(h.RepositoryServerSecrets.repositoryPassword.Data[repoPasswordKey]),
			ConfigFilePath: command.DefaultConfigFilePath,
			LogDirectory:   command.DefaultCacheDirectory,
		},
		CacheDirectory:  defaultCacheDirectory,
		Hostname:        h.RepositoryServer.Spec.Repository.Hostname,
		ContentCacheMB:  contentCacheMB,
		MetadataCacheMB: metadataCacheMB,
		Username:        h.RepositoryServer.Spec.Repository.Username,
		// TODO(Amruta): Generate path for respository
		RepoPathPrefix: h.RepositoryServer.Spec.Repository.RootPath,
		Location:       h.RepositoryServerSecrets.storage.Data,
	}

	return repository.ConnectToKopiaRepository(h.KubeCli, h.RepositoryServer.Namespace, h.RepositoryServer.Status.ServerInfo.PodName, repoServerPodContainerName, args)

}

func (h *RepoServerHandler) getRepositoryCacheSettings() (contentCacheMB, metadataCacheMB int, err error) {
	contentCacheMB, metadataCacheMB = command.GetGeneralCacheSizeSettings()
	if h.RepositoryServer.Spec.Repository.CacheSizeSettings.Content != "" {
		contentCacheMB, err = strconv.Atoi(h.RepositoryServer.Spec.Repository.CacheSizeSettings.Content)
		if err != nil {
			return
		}
	}
	if h.RepositoryServer.Spec.Repository.CacheSizeSettings.Metadata != "" {
		contentCacheMB, err = strconv.Atoi(h.RepositoryServer.Spec.Repository.CacheSizeSettings.Metadata)
		if err != nil {
			return
		}
	}
	return
}
