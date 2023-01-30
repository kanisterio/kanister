package repositoryserver

import (
	"github.com/kanisterio/kanister/pkg/kopia"
	"github.com/kanisterio/kanister/pkg/kopia/command"
	"github.com/kanisterio/kanister/pkg/kopia/repository"
)

const (
	repoPasswordKey           = "repo-password"
	defaultRepoConfigFilePath = "/tmp/config.file"
	defaultRepoLogDirectory   = "/tmp/log.dir"
	defaultCacheDirectory     = "/tmp/cache.dir"
)

func (h *RepoServerHandler) ConnectKopiaRepository() error {

	args := command.RepositoryCommandArgs{
		CommandArgs: &command.CommandArgs{
			RepoPassword:   string(h.RepositoryServerSecrets.repositoryPassword.Data[repoPasswordKey]),
			ConfigFilePath: command.DefaultConfigFilePath,
			LogDirectory:   command.DefaultCacheDirectory,
		},
		CacheDirectory: defaultCacheDirectory,
		Hostname:       h.RepositoryServer.Spec.Repository.Hostname,
		// TODO(Amruta) : check what should be contentCacheMB,metadataCacheMB
		ContentCacheMB:  kopia.GetDataStoreGeneralContentCacheSize(nil),
		MetadataCacheMB: kopia.GetDataStoreGeneralMetadataCacheSize(nil),
		Username:        h.RepositoryServer.Spec.Repository.Username,
		// TODO(Amruta): Generate path for respository
		RepoPathPrefix: h.RepositoryServer.Spec.Repository.RootPath,
		Location:       h.RepositoryServerSecrets.storage.Data,
	}

	return repository.ConnectToKopiaRepository(h.KubeCli, h.RepositoryServer.Namespace, h.RepositoryServer.Status.ServerInfo.PodName, repoServerPodContainerName, args)

}
