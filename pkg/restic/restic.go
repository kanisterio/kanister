package restic

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/location"
	"github.com/kanisterio/kanister/pkg/param"
)

func shCommand(command string) []string {
	return []string{"sh", "-o", "errexit", "-o", "pipefail", "-c", command}
}

// BackupCommand returns restic backup command
func BackupCommand(profile *param.Profile, repository, id, includePath, encryptionKey string) []string {
	cmd := resticArgs(profile, repository, encryptionKey)
	cmd = append(cmd, "backup")
	command := strings.Join(cmd, " ")
	command = fmt.Sprintf("%s --tag %s %s", command, id, includePath)
	return shCommand(command)
}

// RestoreCommand returns restic restore command
func RestoreCommand(profile *param.Profile, repository, id, restorePath, encryptionKey string) []string {
	cmd := resticArgs(profile, repository, encryptionKey)
	cmd = append(cmd, "restore")
	command := strings.Join(cmd, " ")
	command = fmt.Sprintf("%s --tag %s latest --target %s", command, id, restorePath)
	return shCommand(command)
}

// SnapshotsCommand returns restic snapshots command
func SnapshotsCommand(profile *param.Profile, repository, encryptionKey string) []string {
	cmd := resticArgs(profile, repository, encryptionKey)
	cmd = append(cmd, "snapshots")
	command := strings.Join(cmd, " ")
	return shCommand(command)
}

// InitCommand returns restic init command
func InitCommand(profile *param.Profile, repository, encryptionKey string) []string {
	cmd := resticArgs(profile, repository, encryptionKey)
	cmd = append(cmd, "init")
	command := strings.Join(cmd, " ")
	return shCommand(command)
}

// ForgetCommand returns restic forget command
func ForgetCommand(profile *param.Profile, repository, encryptionKey string) []string {
	cmd := resticArgs(profile, repository, encryptionKey)
	cmd = append(cmd, "forget")
	command := strings.Join(cmd, " ")
	return shCommand(command)
}

// PruneCommand returns restic prune command
func PruneCommand(profile *param.Profile, repository, encryptionKey string) []string {
	cmd := resticArgs(profile, repository, encryptionKey)
	cmd = append(cmd, "prune")
	command := strings.Join(cmd, " ")
	return shCommand(command)
}

const (
	ResticPassword   = "RESTIC_PASSWORD"
	ResticRepository = "RESTIC_REPOSITORY"
	ResticCommand    = "restic"
	awsS3Endpoint    = "s3.amazonaws.com"
)

func resticArgs(profile *param.Profile, repository, encryptionKey string) []string {
	s3Endpoint := awsS3Endpoint
	if profile.Location.Endpoint != "" {
		s3Endpoint = profile.Location.Endpoint
	}
	return []string{
		fmt.Sprintf("export %s=%s\n", location.AWSAccessKeyID, profile.Credential.KeyPair.ID),
		fmt.Sprintf("export %s=%s\n", location.AWSSecretAccessKey, profile.Credential.KeyPair.Secret),
		fmt.Sprintf("export %s=%s\n", ResticPassword, encryptionKey),
		fmt.Sprintf("export %s=s3:%s/%s\n", ResticRepository, s3Endpoint, repository),
		ResticCommand,
	}
}

// GetOrCreateRepository will check if the repository already exists and initialize one if not
func GetOrCreateRepository(cli kubernetes.Interface, namespace, pod, container, artifactPrefix, encryptionKey string, profile *param.Profile) error {
	// Use the snapshots command to check if the repository exists
	cmd := SnapshotsCommand(profile, artifactPrefix, encryptionKey)
	stdout, stderr, err := kube.Exec(cli, namespace, pod, container, cmd)
	format.Log(pod, container, stdout)
	format.Log(pod, container, stderr)
	if err == nil {
		return nil
	}
	// Create a repository
	cmd = InitCommand(profile, artifactPrefix, encryptionKey)
	stdout, stderr, err = kube.Exec(cli, namespace, pod, container, cmd)
	format.Log(pod, container, stdout)
	format.Log(pod, container, stderr)
	return errors.Wrapf(err, "Failed to create object store backup location")
}
