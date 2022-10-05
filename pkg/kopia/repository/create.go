package repository

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kopia/command"
	kerrors "github.com/kanisterio/kanister/pkg/kopia/errors"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/utils"
)

const (
	// Kopia returned errors
	errCodeOutOfMemory = "command terminated with exit code 137"
	errAccessDenied    = "Access Denied"
	errInvalidPassword = "invalid repository password"
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
	cmd, err := command.RepositoryCreateCommand(cmdArgs)
	if err != nil {
		return errors.Wrap(err, "Failed to generate repository create command")
	}
	stdout, stderr, err := kube.Exec(cli, namespace, pod, container, cmd, nil)
	format.Log(pod, container, stdout)
	format.Log(pod, container, stderr)

	message := "Failed to create the backup repository"
	switch {
	case err != nil && strings.Contains(err.Error(), errCodeOutOfMemory):
		message = message + ": " + kerrors.ErrOutOfMemoryStr
	case strings.Contains(stderr, errAccessDenied):
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
	mods := getPolicyModifications()
	cmd := command.PolicySetGlobal(command.PolicySetGlobalCommandArgs{CommandArgs: cmdArgs, Modifications: mods})
	stdout, stderr, err := kube.Exec(cli, namespace, pod, container, cmd, nil)
	format.Log(pod, container, stdout)
	format.Log(pod, container, stderr)
	return err
}

// List of possible modifications to a policy, expressed as the kopia flag that will modify it
const (
	// Retention
	keepLatest  = "--keep-latest"
	keepHourly  = "--keep-hourly"
	keepDaily   = "--keep-daily"
	keepWeekly  = "--keep-weekly"
	keepMonthly = "--keep-monthly"
	keepAnnual  = "--keep-annual"

	// Compression
	compressionAlgorithm = "--compression"

	// Compression Algorithms recognized by Kopia
	s2DefaultComprAlgo = "s2-default"
)

func getPolicyModifications() map[string]string {
	const maxInt32 = 1<<31 - 1

	pc := map[string]string{
		// Retention changes
		keepLatest:  strconv.Itoa(maxInt32),
		keepHourly:  strconv.Itoa(0),
		keepDaily:   strconv.Itoa(0),
		keepWeekly:  strconv.Itoa(0),
		keepMonthly: strconv.Itoa(0),
		keepAnnual:  strconv.Itoa(0),

		// Compression changes
		compressionAlgorithm: s2DefaultComprAlgo,
	}
	return pc
}

const (
	MaintenanceOwnerFormat = "%s@%s-maintenance"
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
	newOwner := fmt.Sprintf(MaintenanceOwnerFormat, username, nsUID)
	cmd := command.MaintenanceSetOwner(command.MaintenanceSetOwnerCommandArgs{
		CommandArgs: cmdArgs,
		CustomOwner: newOwner,
	})
	stdout, stderr, err := kube.Exec(cli, namespace, pod, container, cmd, nil)
	format.Log(pod, container, stdout)
	format.Log(pod, container, stderr)
	return err
}
