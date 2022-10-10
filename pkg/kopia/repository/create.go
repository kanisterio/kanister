package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kopia/command"
	"github.com/kanisterio/kanister/pkg/kopia/command/storage"
	kerrors "github.com/kanisterio/kanister/pkg/kopia/errors"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/utils"
)

// CreateKopiaRepository creates a kopia repository if not already present
// Returns true if successful or false with an error
// If the error is an already exists error, returns false with no error
func CreateKopiaRepository(
	cli kubernetes.Interface,
	namespace,
	pod,
	container string,
	cmdArgs command.RepositoryCommandArgs,
) error {
	loc, creds, err := getLocationAndCredsFromMountPath(cli, namespace, pod, container)
	if err != nil {
		return errors.Wrap(err, "Failed to create Kopia repository")
	}
	cmdArgs.Location = loc
	cmdArgs.Credentials = creds
	cmd, err := command.RepositoryCreateCommand(cmdArgs)
	if err != nil {
		return errors.Wrap(err, "Failed to generate repository create command")
	}
	stdout, stderr, err := kube.Exec(cli, namespace, pod, container, cmd, nil)
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
		return errors.Wrap(err, message)
	}

	if err := setGlobalPolicy(
		cli,
		namespace,
		pod,
		container,
		cmdArgs.CommandArgs,
	); err != nil {
		return errors.Wrap(err, "Failed to set global policy")
	}

	// Set custom maintenance owner in case of successful repository creation
	if err := setCustomMaintenanceOwner(
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
func setGlobalPolicy(cli kubernetes.Interface, namespace, pod, container string, cmdArgs *command.CommandArgs) error {
	mods := command.GetPolicyModifications()
	cmd := command.PolicySetGlobal(command.PolicySetGlobalCommandArgs{CommandArgs: cmdArgs, Modifications: mods})
	stdout, stderr, err := kube.Exec(cli, namespace, pod, container, cmd, nil)
	format.Log(pod, container, stdout)
	format.Log(pod, container, stderr)
	return err
}

const (
	maintenanceOwnerFormat = "%s@%s-maintenance"
)

// setCustomMaintenanceOwner sets custom maintenance owner as username@NSUID-maintenance
func setCustomMaintenanceOwner(
	cli kubernetes.Interface,
	namespace,
	pod,
	container,
	username string,
	cmdArgs *command.CommandArgs,
) error {
	nsUID, err := utils.GetNamespaceUID(context.Background(), cli, namespace)
	if err != nil {
		return errors.Wrap(err, "Failed to get namespace UID")
	}
	newOwner := fmt.Sprintf(maintenanceOwnerFormat, username, nsUID)
	cmd := command.MaintenanceSetOwner(command.MaintenanceSetOwnerCommandArgs{
		CommandArgs: cmdArgs,
		CustomOwner: newOwner,
	})
	stdout, stderr, err := kube.Exec(cli, namespace, pod, container, cmd, nil)
	format.Log(pod, container, stdout)
	format.Log(pod, container, stderr)
	return err
}

func getLocationAndCredsFromMountPath(
	cli kubernetes.Interface,
	namespace,
	pod,
	container string,
) (map[string]string, map[string]string, error) {
	fr := kube.NewPodFileReader(cli, pod, namespace, container)
	loc, err := fr.ReadDir(context.Background(), LocationSecretMountPath)
	if err != nil {
		return nil, nil, err
	}
	if storage.SkipCredentialSecretMount(loc) {
		return loc, nil, err
	}
	creds, err := fr.ReadDir(context.Background(), CredsSecretMountPath)
	if err != nil {
		return nil, nil, err
	}
	return loc, creds, nil
}
