// Copyright 2022 The Kanister Authors.
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

package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kopia"
	kopiacmd "github.com/kanisterio/kanister/pkg/kopia/command"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	kanutils "github.com/kanisterio/kanister/pkg/utils"
)

const (
	MaintenanceOwnerFormat = "%s@%s-maintenance"
)

// CreateKopiaRepository creates a kopia repository if not already present
// Returns true if successful or false with an error
// If the error is an already exists error, returns false with no error
func CreateKopiaRepository(
	cli kubernetes.Interface,
	namespace,
	pod,
	container,
	artifactPrefix,
	encryptionKey,
	hostname,
	username,
	cacheDirectory,
	configFilePath,
	logDirectory string,
	contentCacheMB,
	metadataCacheMB int,
	prof kopia.Profile,
) error {
	args := kopiacmd.RepositoryCreateCommandArgs{
		CommandArgs: &kopiacmd.CommandArgs{
			EncryptionKey:  encryptionKey,
			ConfigFilePath: configFilePath,
			LogDirectory:   logDirectory,
		},
		Prof:            prof,
		ArtifactPrefix:  artifactPrefix,
		Hostname:        hostname,
		Username:        username,
		CacheDirectory:  cacheDirectory,
		ContentCacheMB:  contentCacheMB,
		MetadataCacheMB: metadataCacheMB,
	}
	cmd, err := kopiacmd.RepositoryCreate(args)
	if err != nil {
		return errors.Wrap(err, "Failed to generate repository create command")
	}
	stdout, stderr, err := kube.Exec(cli, namespace, pod, container, cmd, nil)
	format.Log(pod, container, stdout)
	format.Log(pod, container, stderr)

	message := "Failed to create the backup repository"
	switch {
	case err != nil && strings.Contains(err.Error(), kopia.ErrCodeOutOfMemory.Error()):
		message = message + ": " + kopia.ErrOutOfMemory.Error()
	case strings.Contains(stderr, kopia.ErrAccessDenied.Error()):
		message = message + ": " + kopia.ErrAccessDenied.Error()
	}
	if err != nil {
		return errors.Wrap(err, message)
	}

	if err := setGlobalPolicy(
		cli,
		namespace,
		pod,
		container,
		artifactPrefix,
		encryptionKey,
		configFilePath,
		logDirectory,
	); err != nil {
		return errors.Wrap(err, "Failed to set global policy")
	}

	// Set custom maintenance owner in case of successful repository creation
	if err := setCustomMaintenanceOwner(
		cli,
		namespace,
		pod,
		container,
		artifactPrefix,
		encryptionKey,
		hostname,
		username,
		configFilePath,
		logDirectory,
	); err != nil {
		log.WithError(err).Print("Failed to set custom kopia maintenance owner, proceeding with default owner")
	}
	return nil
}

// setGlobalPolicy sets the global policy of the kopia repo to keep max-int32 latest
// snapshots and zeros all other time-based retention fields
func setGlobalPolicy(cli kubernetes.Interface, namespace, pod, container, artifactPrefix, encryptionKey, configFilePath, logDirectory string) error {
	cmd := kopia.PolicySetGlobalCommand(encryptionKey, configFilePath, logDirectory)
	stdout, stderr, err := kube.Exec(cli, namespace, pod, container, cmd, nil)
	format.Log(pod, container, stdout)
	format.Log(pod, container, stderr)
	return err
}

// setCustomMaintenanceOwner sets custom maintenance owner as hostname@NSUID-maintenance
func setCustomMaintenanceOwner(
	cli kubernetes.Interface,
	namespace,
	pod,
	container,
	artifactPrefix,
	encryptionKey,
	hostname,
	username,
	configFilePath,
	logDirectory string,
) error {
	nsUID, err := kanutils.GetNamespaceUID(context.Background(), cli, namespace)
	if err != nil {
		return errors.Wrap(err, "Failed to get namespace UID")
	}
	newOwner := fmt.Sprintf(MaintenanceOwnerFormat, username, nsUID)
	args := kopiacmd.MaintenanceSetOwnerCommandArgs{
		CommandArgs: &kopiacmd.CommandArgs{
			EncryptionKey:  encryptionKey,
			ConfigFilePath: configFilePath,
			LogDirectory:   logDirectory,
		},
		CustomOwner: newOwner,
	}
	cmd := kopiacmd.MaintenanceSetOwner(args)
	stdout, stderr, err := kube.Exec(cli, namespace, pod, container, cmd, nil)
	format.Log(pod, container, stdout)
	format.Log(pod, container, stderr)
	return err
}
