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

	"github.com/kanisterio/errkit"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kopia/command"
	kerrors "github.com/kanisterio/kanister/pkg/kopia/errors"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/utils"
)

// CreateKopiaRepository creates a kopia repository if not already present
// Returns true if successful or false with an error
// If the error is an already exists error, returns false with no error
func CreateKopiaRepository(
	ctx context.Context,
	cli kubernetes.Interface,
	namespace,
	pod,
	container string,
	cmdArgs command.RepositoryCommandArgs,
) error {
	cmd, err := command.RepositoryCreateCommand(cmdArgs)
	if err != nil {
		return errkit.Wrap(err, "Failed to generate repository create command")
	}
	stdout, stderr, err := kube.Exec(ctx, cli, namespace, pod, container, cmd, nil)
	format.Log(pod, container, stdout)
	format.Log(pod, container, stderr)

	message := "Failed to create the backup repository"
	switch {
	case err != nil && strings.Contains(err.Error(), kerrors.ErrCodeOutOfMemoryStr):
		message = message + ": " + kerrors.ErrOutOfMemoryStr
	case strings.Contains(stderr, kerrors.ErrAccessDeniedStr):
		message = message + ": " + kerrors.ErrAccessDeniedStr
	}
	if err != nil {
		return errkit.Wrap(err, message)
	}

	if err := setGlobalPolicy(
		ctx,
		cli,
		namespace,
		pod,
		container,
		cmdArgs.CommandArgs,
	); err != nil {
		return errkit.Wrap(err, "Failed to set global policy")
	}

	// Set custom maintenance owner in case of successful repository creation
	if err := setCustomMaintenanceOwner(
		ctx,
		cli,
		namespace,
		pod,
		container,
		cmdArgs.Username,
		cmdArgs.CommandArgs,
	); err != nil {
		log.WithError(err).Print("Failed to set custom kopia maintenance owner, proceeding with default owner")
	}
	return nil
}

// setGlobalPolicy sets the global policy of the kopia repo to keep max-int32 latest
// snapshots and zeros all other time-based retention fields
func setGlobalPolicy(
	ctx context.Context,
	cli kubernetes.Interface,
	namespace,
	pod,
	container string,
	cmdArgs *command.CommandArgs,
) error {
	mods := command.GetPolicyModifications()
	cmd := command.PolicySetGlobal(command.PolicySetGlobalCommandArgs{CommandArgs: cmdArgs, Modifications: mods})
	stdout, stderr, err := kube.Exec(ctx, cli, namespace, pod, container, cmd, nil)
	format.Log(pod, container, stdout)
	format.Log(pod, container, stderr)
	return err
}

const (
	maintenanceOwnerFormat = "%s@%s-maintenance"
)

// setCustomMaintenanceOwner sets custom maintenance owner as username@NSUID-maintenance
func setCustomMaintenanceOwner(
	ctx context.Context,
	cli kubernetes.Interface,
	namespace,
	pod,
	container,
	username string,
	cmdArgs *command.CommandArgs,
) error {
	nsUID, err := utils.GetNamespaceUID(context.Background(), cli, namespace)
	if err != nil {
		return errkit.Wrap(err, "Failed to get namespace UID")
	}
	newOwner := fmt.Sprintf(maintenanceOwnerFormat, username, nsUID)
	cmd := command.MaintenanceSetOwner(command.MaintenanceSetOwnerCommandArgs{
		CommandArgs: cmdArgs,
		CustomOwner: newOwner,
	})
	stdout, stderr, err := kube.Exec(ctx, cli, namespace, pod, container, cmd, nil)
	format.Log(pod, container, stdout)
	format.Log(pod, container, stderr)
	return err
}
