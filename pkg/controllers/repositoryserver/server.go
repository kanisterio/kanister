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
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	"k8s.io/kube-openapi/pkg/util/sets"

	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kopia"
	"github.com/kanisterio/kanister/pkg/kopia/command"
	"github.com/kanisterio/kanister/pkg/kopia/maintenance"
	"github.com/kanisterio/kanister/pkg/kube"
	reposerver "github.com/kanisterio/kanister/pkg/secrets/repositoryserver"
)

const (
	// DefaultServerStartTimeout is default time to create context for Kopia API server Status Command
	DefaultServerStartTimeout = 600 * time.Second
)

func (h *RepoServerHandler) startRepoProxyServer(ctx context.Context) (err error) {
	repoServerAddress, serverAdminUserName, serverAdminPassword, err := h.getServerDetails(ctx)
	if err != nil {
		return err
	}

	err = h.checkServerStatus(ctx, repoServerAddress, serverAdminUserName, serverAdminPassword)
	if err == nil {
		h.Logger.Info("Kopia API server already started")
		return nil
	}

	cmd := command.ServerStart(
		command.ServerStartCommandArgs{
			CommandArgs: &command.CommandArgs{
				RepoPassword:   "",
				ConfigFilePath: command.DefaultConfigFilePath,
				LogDirectory:   command.DefaultLogDirectory,
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
	stdout, stderr, err := kube.Exec(ctx, h.KubeCli, h.RepositoryServer.Namespace, h.RepositoryServer.Status.ServerInfo.PodName, repoServerPodContainerName, cmd, nil)
	format.Log(h.RepositoryServer.Status.ServerInfo.PodName, repoServerPodContainerName, stdout)
	format.Log(h.RepositoryServer.Status.ServerInfo.PodName, repoServerPodContainerName, stderr)
	if err != nil {
		return errors.Wrap(err, "Failed to start Kopia API server")
	}

	err = h.waitForServerReady(ctx, repoServerAddress, serverAdminUserName, serverAdminPassword)
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
	if serverAdminUsername, ok = h.RepositoryServerSecrets.serverAdmin.Data[reposerver.AdminUsernameKey]; !ok {
		return "", "", "", errors.New("Server admin username is not specified")
	}
	if serverAdminPassword, ok = h.RepositoryServerSecrets.serverAdmin.Data[reposerver.AdminPasswordKey]; !ok {
		return "", "", "", errors.New("Server admin password is not specified")
	}
	return repoServerAddress, string(serverAdminUsername), string(serverAdminPassword), nil
}

func (h *RepoServerHandler) checkServerStatus(ctx context.Context, serverAddress, username, password string) error {
	cmd, err := h.getServerStatusCommand(ctx, serverAddress, username, password)
	if err != nil {
		return errors.Wrap(err, "Failed to extract fingerprint from Kopia API server certificate secret data")
	}
	stdout, stderr, exErr := kube.Exec(ctx, h.KubeCli, h.RepositoryServer.Namespace, h.RepositoryServer.Status.ServerInfo.PodName, repoServerPodContainerName, cmd, nil)
	format.Log(h.RepositoryServer.Status.ServerInfo.PodName, repoServerPodContainerName, stdout)
	format.Log(h.RepositoryServer.Status.ServerInfo.PodName, repoServerPodContainerName, stderr)
	return exErr
}

func (h *RepoServerHandler) waitForServerReady(ctx context.Context, serverAddress, username, password string) error {
	cmd, err := h.getServerStatusCommand(ctx, serverAddress, username, password)
	if err != nil {
		return errors.Wrap(err, "Failed to extract fingerprint from Kopia API server certificate secret data")
	}
	serverStartTimeOut := h.getRepositoryServerStartTimeout()
	ctx, cancel := context.WithTimeout(ctx, serverStartTimeOut)
	defer cancel()
	return WaitTillCommandSucceed(ctx, h.KubeCli, cmd, h.RepositoryServer.Namespace, h.RepositoryServer.Status.ServerInfo.PodName, repoServerPodContainerName)
}

func (h *RepoServerHandler) createOrUpdateClientUsers(ctx context.Context) error {
	repoPassword := string(h.RepositoryServerSecrets.repositoryPassword.Data[reposerver.RepoPasswordKey])

	cmd := command.ServerListUser(
		command.ServerListUserCommmandArgs{
			CommandArgs: &command.CommandArgs{
				RepoPassword:   repoPassword,
				ConfigFilePath: command.DefaultConfigFilePath,
				LogDirectory:   command.DefaultLogDirectory,
			},
		})
	stdout, stderr, err := kube.Exec(ctx, h.KubeCli, h.RepositoryServer.Namespace, h.RepositoryServer.Status.ServerInfo.PodName, repoServerPodContainerName, cmd, nil)
	format.Log(h.RepositoryServer.Status.ServerInfo.PodName, repoServerPodContainerName, stdout)
	format.Log(h.RepositoryServer.Status.ServerInfo.PodName, repoServerPodContainerName, stderr)
	if err != nil {
		return errors.Wrap(err, "Failed to list users from the Kopia repository")
	}

	userProfiles := []maintenance.KopiaUserProfile{}

	err = json.Unmarshal([]byte(stdout), &userProfiles)
	if err != nil {
		return errors.Wrap(err, "Failed to unmarshal user list")
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
			stdout, stderr, err := kube.Exec(ctx, h.KubeCli, h.RepositoryServer.Namespace, h.RepositoryServer.Status.ServerInfo.PodName, repoServerPodContainerName, cmd, nil)
			format.Log(h.RepositoryServer.Status.ServerInfo.PodName, repoServerPodContainerName, stdout)
			format.Log(h.RepositoryServer.Status.ServerInfo.PodName, repoServerPodContainerName, stderr)
			if err != nil {
				return errors.Wrap(err, "Failed to update existing user passphrase from the Kopia API server")
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
		stdout, stderr, err := kube.Exec(ctx, h.KubeCli, h.RepositoryServer.Namespace, h.RepositoryServer.Status.ServerInfo.PodName, repoServerPodContainerName, cmd, nil)
		format.Log(h.RepositoryServer.Status.ServerInfo.PodName, repoServerPodContainerName, stdout)
		format.Log(h.RepositoryServer.Status.ServerInfo.PodName, repoServerPodContainerName, stderr)
		if err != nil {
			return errors.Wrap(err, "Failed to add new user to the Kopia API server")
		}
	}
	return nil
}

func (h *RepoServerHandler) refreshServer(ctx context.Context) error {
	serverAddress, username, password, err := h.getServerDetails(ctx)
	if err != nil {
		return err
	}
	repoPassword := string(h.RepositoryServerSecrets.repositoryPassword.Data[reposerver.RepoPasswordKey])
	fingerprint, err := kopia.ExtractFingerprintFromCertSecret(ctx, h.KubeCli, h.RepositoryServerSecrets.serverTLS.Name, h.RepositoryServer.Namespace)
	if err != nil {
		return errors.Wrap(err, "Failed to extract fingerprint from Kopia API server certificate secret data")
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
	stdout, stderr, err := kube.Exec(ctx, h.KubeCli, h.RepositoryServer.Namespace, h.RepositoryServer.Status.ServerInfo.PodName, repoServerPodContainerName, cmd, nil)
	format.Log(h.RepositoryServer.Status.ServerInfo.PodName, repoServerPodContainerName, stdout)
	format.Log(h.RepositoryServer.Status.ServerInfo.PodName, repoServerPodContainerName, stderr)
	if err != nil {
		return errors.Wrap(err, "Failed to refresh Kopia API server")
	}
	return nil
}

func (h *RepoServerHandler) getRepositoryServerStartTimeout() time.Duration {
	serverStartTimeoutEnv := os.Getenv("KOPIA_SERVER_START_TIMEOUT")
	if serverStartTimeoutEnv != "" {
		serverStartTimeout, err := time.ParseDuration(serverStartTimeoutEnv)
		if err != nil {
			h.Logger.Info("Error parsing env variable", "error", err)
			return DefaultServerStartTimeout
		}
		return serverStartTimeout * time.Second
	}
	return DefaultServerStartTimeout
}

func (h *RepoServerHandler) getServerStatusCommand(ctx context.Context, serverAddress, username, password string) ([]string, error) {
	fingerprint, err := kopia.ExtractFingerprintFromCertSecret(ctx, h.KubeCli, h.RepositoryServerSecrets.serverTLS.Name, h.RepositoryServer.Namespace)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to extract fingerprint from Kopia API server certificate secret data")
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
	return cmd, nil
}
