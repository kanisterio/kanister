package function

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/restic"
)

const (
	// BackupDataNamespaceArg provides the namespace
	BackupDataNamespaceArg = "namespace"
	// BackupDataPodArg provides the pod connected to the data volume
	BackupDataPodArg = "pod"
	// BackupDataContainerArg provides the container on which the backup is taken
	BackupDataContainerArg = "container"
	// BackupDataIncludePathArg provides the path of the volume or sub-path for required backup
	BackupDataIncludePathArg = "includePath"
	// BackupDataBackupArtifactArg provides the path to store artifacts on the object store
	BackupDataBackupArtifactArg = "backupArtifact"
	// BackupDataBackupTagArg provides a tag to be added to the artifacts
	BackupDataBackupTagArg = "backupTag"
)

func init() {
	kanister.Register(&backupDataFunc{})
}

var _ kanister.Func = (*backupDataFunc)(nil)

type backupDataFunc struct{}

func (*backupDataFunc) Name() string {
	return "BackupData"
}

func generateSnapshotsCommand(destArtifact string, profile *param.Profile) []string {
	// Restic Snapshots command
	command := restic.SnapshotsCommand(profile, destArtifact)
	return []string{"sh", "-o", "errexit", "-o", "pipefail", "-c", command}
}

func generateInitCommand(destArtifact string, profile *param.Profile) []string {
	// Restic Repository Init command
	command := restic.InitCommand(profile, destArtifact)
	return []string{"sh", "-o", "errexit", "-o", "pipefail", "-c", command}
}

func generateBackupCommand(includePath, destArtifact, tag string, profile *param.Profile) []string {
	// Restic Backup command
	command := restic.BackupCommand(profile, destArtifact)
	command = fmt.Sprintf("%s --tag %s %s", command, tag, includePath)
	return []string{"sh", "-o", "errexit", "-o", "pipefail", "-c", command}
}

func validateProfile(profile *param.Profile) error {
	if profile == nil {
		return errors.New("Profile must be non-nil")
	}
	if profile.Location.Type != crv1alpha1.LocationTypeS3Compliant {
		return errors.New("Location type not supported")
	}
	return nil
}

func (*backupDataFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) error {
	var namespace, pod, container, includePath, backupArtifact, backupTag string
	var err error
	if err = Arg(args, BackupDataNamespaceArg, &namespace); err != nil {
		return err
	}
	if err = Arg(args, BackupDataPodArg, &pod); err != nil {
		return err
	}
	if err = Arg(args, BackupDataContainerArg, &container); err != nil {
		return err
	}
	if err = Arg(args, BackupDataIncludePathArg, &includePath); err != nil {
		return err
	}
	if err = Arg(args, BackupDataBackupArtifactArg, &backupArtifact); err != nil {
		return err
	}
	// TODO: Change this to required arg once all the changes are done
	if err = OptArg(args, BackupDataBackupTagArg, &backupTag, time.Now().UnixNano()); err != nil {
		return err
	}
	// Validate the Profile
	if err = validateProfile(tp.Profile); err != nil {
		return errors.Wrapf(err, "Failed to validate Profile")
	}
	cli, err := kube.NewClient()
	if err != nil {
		return errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	// Use the snapshots command to check if the repository exists
	cmd := generateSnapshotsCommand(backupArtifact, tp.Profile)
	stdout, stderr, err := kube.Exec(cli, namespace, pod, container, cmd)
	formatAndLog(pod, container, stdout)
	formatAndLog(pod, container, stderr)
	if err != nil {
		// Create a repository
		cmd := generateInitCommand(backupArtifact, tp.Profile)
		stdout, stderr, err := kube.Exec(cli, namespace, pod, container, cmd)
		formatAndLog(pod, container, stdout)
		formatAndLog(pod, container, stderr)
		if err != nil {
			return errors.Wrapf(err, "Failed to create object store backup location")
		}
	}
	// Create backup and dump it on the object store
	cmd = generateBackupCommand(includePath, backupArtifact, backupTag, tp.Profile)
	stdout, stderr, err = kube.Exec(cli, namespace, pod, container, cmd)
	formatAndLog(pod, container, stdout)
	formatAndLog(pod, container, stderr)
	if err != nil {
		return errors.Wrapf(err, "Failed to create and upload backup")
	}
	return nil
}

func (*backupDataFunc) RequiredArgs() []string {
	return []string{BackupDataNamespaceArg, BackupDataPodArg, BackupDataContainerArg,
		BackupDataIncludePathArg, BackupDataBackupArtifactArg}
}
