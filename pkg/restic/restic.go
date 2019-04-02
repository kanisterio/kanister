package restic

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/location"
	"github.com/kanisterio/kanister/pkg/param"
)

func shCommand(command string) []string {
	return []string{"bash", "-o", "errexit", "-o", "pipefail", "-c", command}
}

// BackupCommandByID returns restic backup command
func BackupCommandByID(profile *param.Profile, repository, pathToBackup, encryptionKey string) []string {
	cmd := resticArgs(profile, repository, encryptionKey)
	cmd = append(cmd, "backup", pathToBackup)
	command := strings.Join(cmd, " ")
	return shCommand(command)
}

// BackupCommandByTag returns restic backup command with tag
func BackupCommandByTag(profile *param.Profile, repository, backupTag, includePath, encryptionKey string) []string {
	cmd := resticArgs(profile, repository, encryptionKey)
	cmd = append(cmd, "backup", "--tag", backupTag, includePath)
	command := strings.Join(cmd, " ")
	return shCommand(command)
}

// RestoreCommandByID returns restic restore command with snapshotID as the identifier
func RestoreCommandByID(profile *param.Profile, repository, id, restorePath, encryptionKey string) []string {
	cmd := resticArgs(profile, repository, encryptionKey)
	cmd = append(cmd, "restore", id, "--target", restorePath)
	command := strings.Join(cmd, " ")
	return shCommand(command)
}

// RestoreCommandByTag returns restic restore command with tag as the identifier
func RestoreCommandByTag(profile *param.Profile, repository, tag, restorePath, encryptionKey string) []string {
	cmd := resticArgs(profile, repository, encryptionKey)
	cmd = append(cmd, "restore", "--tag", tag, "latest", "--target", restorePath)
	command := strings.Join(cmd, " ")
	return shCommand(command)
}

// SnapshotsCommand returns restic snapshots command
func SnapshotsCommand(profile *param.Profile, repository, encryptionKey string) []string {
	cmd := resticArgs(profile, repository, encryptionKey)
	cmd = append(cmd, "snapshots")
	command := strings.Join(cmd, " ")
	return shCommand(command)
}

// SnapshotsCommandByTag returns restic snapshots command
func SnapshotsCommandByTag(profile *param.Profile, repository, tag, encryptionKey string) []string {
	cmd := resticArgs(profile, repository, encryptionKey)
	cmd = append(cmd, "snapshots", "--tag", tag, "--json")
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

// ForgetCommandByTag returns restic forget command
func ForgetCommandByTag(profile *param.Profile, repository, tag, encryptionKey string) []string {
	cmd := resticArgs(profile, repository, encryptionKey)
	cmd = append(cmd, "forget", "--tag", tag)
	command := strings.Join(cmd, " ")
	return shCommand(command)
}

// ForgetCommandByID returns restic forget command
func ForgetCommandByID(profile *param.Profile, repository, id, encryptionKey string) []string {
	cmd := resticArgs(profile, repository, encryptionKey)
	cmd = append(cmd, "forget", id)
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

// SnapshotIDFromSnapshotLog gets the SnapshotID from Snapshot Command log
func SnapshotIDFromSnapshotLog(output string) (string, error) {
	var result []map[string]interface{}
	err := json.Unmarshal([]byte(output), &result)
	if err != nil {
		return "", errors.WithMessage(err, "Failed to unmarshall output from snapshotCommand")
	}
	if len(result) != 1 {
		return "", errors.New("Snapshot not found")
	}
	snapId := result[0]["short_id"]
	return snapId.(string), nil
}

// SnapshotIDFromBackupLog gets the SnapshotID from Backup Command log
func SnapshotIDFromBackupLog(output string) string {
	if output == "" {
		return ""
	}
	logs := regexp.MustCompile("[\n]").Split(output, -1)
	for _, l := range logs {
		// Log should contain "snapshot ABC123 saved"
		pattern := regexp.MustCompile(`snapshot\s(.*?)\ssaved$`)
		match := pattern.FindAllStringSubmatch(l, 1)
		if match != nil {
			return match[0][1]
		}
	}
	return ""
}
