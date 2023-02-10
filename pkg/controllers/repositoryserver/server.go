package repositoryserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kopia"
	"github.com/kanisterio/kanister/pkg/kopia/command"
	"github.com/kanisterio/kanister/pkg/kopia/maintenance"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/kyaml/sets"
)

const (
	serverAdminUserNameKey = "username"
	serverAdminPasswordKey = "password"
	// DefaultServerStartTimeout is default time to create context for Kopia API Server Status Command
	DefaultServerStartTimeout = 600 * time.Second
)

func (h *RepoServerHandler) startRepoProxyServer(ctx context.Context) (err error) {
	repoServerAddress, serverAdminUserName, serverAdminPassword, err := h.getServerDetails(ctx)
	if err != nil {
		return err
	}

	err = h.checkServerStatus(ctx, repoServerAddress, serverAdminUserName, serverAdminPassword)
	if err == nil {
		h.Logger.Info("Repository server already started")
		return nil
	}

	cmd := command.ServerStart(
		command.ServerStartCommandArgs{
			CommandArgs: &command.CommandArgs{
				RepoPassword:   "",
				ConfigFilePath: command.DefaultConfigFilePath,
				LogDirectory:   command.DefaultCacheDirectory,
			},
			ServerAddress:    repoServerAddress,
			TLSCertFile:      tlsCertPath,
			TLSKeyFile:       tlsKeyPath,
			ServerUsername:   serverAdminUserName,
			ServerPassword:   serverAdminPassword,
			AutoGenerateCert: false,
			Background:       true,
		},
	)
	stdout, stderr, err := kube.Exec(h.KubeCli, h.RepositoryServer.Namespace, h.RepositoryServer.Status.ServerInfo.PodName, repoServerPodContainerName, cmd, nil)
	format.Log(h.RepositoryServer.Status.ServerInfo.PodName, repoServerPodContainerName, stdout)
	format.Log(h.RepositoryServer.Status.ServerInfo.PodName, repoServerPodContainerName, stderr)
	if err != nil {
		return errors.Wrap(err, "Failed to start Kopia API server")
	}

	err = h.checkServerStatus(ctx, repoServerAddress, serverAdminUserName, serverAdminPassword)
	if err != nil {
		return errors.Wrap(err, "Failed to check Kopia API server status")
	}

	return nil
}

func (h *RepoServerHandler) getServerDetails(ctx context.Context) (string, string, string, error) {
	repoServerAddress, err := getPodAddress(ctx, h.KubeCli, h.RepositoryServer.Namespace, h.RepositoryServer.Status.ServerInfo.PodName)
	if err != nil {
		return "", "", "", err
	}
	var serverAdminUsername, serverAdminPassword []byte
	var ok bool
	if serverAdminUsername, ok = h.RepositoryServerSecrets.serverAdmin.Data[serverAdminUserNameKey]; !ok {
		return "", "", "", errors.New("server admin username is not specified")
	}
	if serverAdminPassword, ok = h.RepositoryServerSecrets.serverAdmin.Data[serverAdminPasswordKey]; !ok {
		return "", "", "", errors.New("server admin password is not specified")
	}
	return repoServerAddress, string(serverAdminUsername), string(serverAdminPassword), nil
}

func (h *RepoServerHandler) checkServerStatus(ctx context.Context, serverAddress, username, password string) error {
	fingerprint, err := kopia.ExtractFingerprintFromCertSecret(ctx, h.KubeCli, h.RepositoryServerSecrets.serverTLS.Name, h.RepositoryServer.Namespace)
	if err != nil {
		return errors.Wrap(err, "Failed to extract fingerprint Kopia API Server Certificate Secret Data")
	}
	cmd := command.ServerStatus(
		command.ServerStatusCommandArgs{
			CommandArgs: &command.CommandArgs{
				RepoPassword:   "",
				ConfigFilePath: command.DefaultConfigFilePath,
				LogDirectory:   command.DefaultLogDirectory,
			},
			ServerAddress:  serverAddress,
			ServerUsername: username,
			ServerPassword: password,
			Fingerprint:    fingerprint,
		})

	serverStartTimeOut, err := h.getRepositoryServerStartTimeout()
	if err != nil {
		return errors.Wrap(err, "failed to get repository server timeout")
	}
	ctx, cancel := context.WithTimeout(ctx, serverStartTimeOut)
	defer cancel()
	return WaitTillCommandSucceed(ctx, h.KubeCli, cmd, h.RepositoryServer.Namespace, h.RepositoryServer.Status.ServerInfo.PodName, repoServerPodContainerName)
}

func (h *RepoServerHandler) createOrUpdateClientUsers(ctx context.Context) error {
	repoPassword := string(h.RepositoryServerSecrets.repositoryPassword.Data[repoPasswordKey])

	cmd := command.ServerListUser(
		command.ServerListUserCommmandArgs{
			CommandArgs: &command.CommandArgs{
				RepoPassword:   repoPassword,
				ConfigFilePath: command.DefaultConfigFilePath,
				LogDirectory:   command.DefaultLogDirectory,
			},
		})
	stdout, stderr, err := kube.Exec(h.KubeCli, h.RepositoryServer.Namespace, h.RepositoryServer.Status.ServerInfo.PodName, repoServerPodContainerName, cmd, nil)
	format.Log(h.RepositoryServer.Status.ServerInfo.PodName, repoServerPodContainerName, stdout)
	format.Log(h.RepositoryServer.Status.ServerInfo.PodName, repoServerPodContainerName, stderr)
	if err != nil {
		errors.Wrap(err, "Failed to list users from the Kopia repository")
	}

	userProfiles := []maintenance.KopiaUserProfile{}

	err = json.Unmarshal([]byte(stdout), &userProfiles)
	if err != nil {
		errors.Wrap(err, "Failed to unmarshal user list")
	}

	// Get list of usernames from ServerListUserCommand output to update the existing data with updated password
	existingUserHostList := sets.String{}
	for _, userProfile := range userProfiles {
		existingUserHostList.Insert(userProfile.Username)
	}

	userAccess := h.RepositoryServerSecrets.serverUserAccess.Data
	serverAccessUsername := h.RepositoryServer.Spec.Server.UserAccess.Username
	for hostname, password := range userAccess {
		serverUsername := fmt.Sprintf(repoServerUsernameFormat, serverAccessUsername, hostname)
		if existingUserHostList.Has(serverUsername) {
			h.Logger.Info("User already exists, updating passphrase", "username", serverUsername)
			// Update password for the existing user
			cmd := command.ServerSetUser(
				command.ServerSetUserCommandArgs{
					CommandArgs: &command.CommandArgs{
						RepoPassword:   repoPassword,
						ConfigFilePath: command.DefaultConfigFilePath,
						LogDirectory:   command.DefaultLogDirectory,
					},
					NewUsername:  serverUsername,
					UserPassword: string(password),
				})
			stdout, stderr, err := kube.Exec(h.KubeCli, h.RepositoryServer.Namespace, h.RepositoryServer.Status.ServerInfo.PodName, repoServerPodContainerName, cmd, nil)
			format.Log(h.RepositoryServer.Status.ServerInfo.PodName, repoServerPodContainerName, stdout)
			format.Log(h.RepositoryServer.Status.ServerInfo.PodName, repoServerPodContainerName, stderr)
			if err != nil {
				errors.Wrap(err, "Failed to update existing user passphrase from the Kopia API server")
			}
			continue
		}
		cmd := command.ServerAddUser(
			command.ServerAddUserCommandArgs{
				CommandArgs: &command.CommandArgs{
					RepoPassword:   repoPassword,
					ConfigFilePath: command.DefaultConfigFilePath,
					LogDirectory:   command.DefaultLogDirectory,
				},
				NewUsername:  serverUsername,
				UserPassword: string(password),
			})
		stdout, stderr, err := kube.Exec(h.KubeCli, h.RepositoryServer.Namespace, h.RepositoryServer.Status.ServerInfo.PodName, repoServerPodContainerName, cmd, nil)
		format.Log(h.RepositoryServer.Status.ServerInfo.PodName, repoServerPodContainerName, stdout)
		format.Log(h.RepositoryServer.Status.ServerInfo.PodName, repoServerPodContainerName, stderr)
		if err != nil {
			return errors.Wrap(err, "Failed to add new user to the Kopia API server")
		}
	}

	repoServerAddress, serverAdminUserName, serverAdminPassword, err := h.getServerDetails(ctx)
	if err != nil {
		return err
	}
	err = h.refreshServer(ctx, repoServerAddress, serverAdminUserName, serverAdminPassword)
	if err != nil {
		return errors.Wrap(err, "Failed to refresh repository server")
	}
	return nil
}

func (h *RepoServerHandler) refreshServer(ctx context.Context, serverAddress, username, password string) error {
	repoPassword := string(h.RepositoryServerSecrets.repositoryPassword.Data[repoPasswordKey])
	fingerprint, err := kopia.ExtractFingerprintFromCertSecret(ctx, h.KubeCli, h.RepositoryServerSecrets.serverTLS.Name, h.RepositoryServer.Namespace)
	if err != nil {
		return errors.Wrap(err, "Failed to extract fingerprint Kopia API Server Certificate Secret Data")
	}

	cmd := command.ServerRefresh(
		command.ServerRefreshCommandArgs{
			CommandArgs: &command.CommandArgs{
				RepoPassword:   repoPassword,
				ConfigFilePath: command.DefaultConfigFilePath,
				LogDirectory:   command.DefaultLogDirectory,
			},
			ServerAddress:  serverAddress,
			ServerUsername: username,
			ServerPassword: password,
			Fingerprint:    fingerprint,
		})
	stdout, stderr, err := kube.Exec(h.KubeCli, h.RepositoryServer.Namespace, h.RepositoryServer.Status.ServerInfo.PodName, repoServerPodContainerName, cmd, nil)
	format.Log(h.RepositoryServer.Status.ServerInfo.PodName, repoServerPodContainerName, stdout)
	format.Log(h.RepositoryServer.Status.ServerInfo.PodName, repoServerPodContainerName, stderr)
	if err != nil {
		errors.Wrap(err, "Failed to refresh Kopia API server")
	}
	return nil
}

func (h *RepoServerHandler) getRepositoryServerStartTimeout() (time.Duration, error) {
	serverStartTimeoutEnv := os.Getenv("KOPIA_SERVER_START_TIMEOUT")
	if serverStartTimeoutEnv != "" {
		serverStartTimeout, err := time.ParseDuration(serverStartTimeoutEnv)
		if err != nil {
			h.Logger.Info("Error parsing env variable", err)
			return DefaultServerStartTimeout, nil
		}
		return serverStartTimeout * time.Second, nil
	}
	return DefaultServerStartTimeout, nil
}
