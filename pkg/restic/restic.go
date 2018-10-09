package restic

import (
	"fmt"
	"strings"

	"github.com/kanisterio/kanister/pkg/location"
	"github.com/kanisterio/kanister/pkg/param"
)

func shCommand(command string) []string {
	return []string{"sh", "-o", "errexit", "-o", "pipefail", "-c", command}
}

// BackupCommand returns restic backup command
func BackupCommand(profile *param.Profile, repository, id, includePath string) []string {
	cmd := resticArgs(profile, repository)
	cmd = append(cmd, "backup")
	command := strings.Join(cmd, " ")
	command = fmt.Sprintf("%s --tag %s %s", command, id, includePath)
	return shCommand(command)
}

// RestoreCommand returns restic restore command
func RestoreCommand(profile *param.Profile, repository, id, restorePath string) []string {
	cmd := resticArgs(profile, repository)
	cmd = append(cmd, "restore")
	command := strings.Join(cmd, " ")
	command = fmt.Sprintf("%s --tag %s latest --target %s", command, id, restorePath)
	return shCommand(command)
}

// SnapshotsCommand returns restic snapshots command
func SnapshotsCommand(profile *param.Profile, repository string) []string {
	cmd := resticArgs(profile, repository)
	cmd = append(cmd, "snapshots")
	command := strings.Join(cmd, " ")
	return shCommand(command)
}

// InitCommand returns restic init command
func InitCommand(profile *param.Profile, repository string) []string {
	cmd := resticArgs(profile, repository)
	cmd = append(cmd, "init")
	command := strings.Join(cmd, " ")
	return shCommand(command)
}

// ForgetCommand returns restic forget command
func ForgetCommand(profile *param.Profile, repository string) []string {
	cmd := resticArgs(profile, repository)
	cmd = append(cmd, "forget")
	command := strings.Join(cmd, " ")
	return shCommand(command)
}

// PruneCommand returns restic prune command
func PruneCommand(profile *param.Profile, repository string) []string {
	cmd := resticArgs(profile, repository)
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

func resticArgs(profile *param.Profile, repository string) []string {
	s3Endpoint := awsS3Endpoint
	if profile.Location.S3Compliant.Endpoint != "" {
		s3Endpoint = profile.Location.S3Compliant.Endpoint
	}
	return []string{
		fmt.Sprintf("export %s=%s\n", location.AWSAccessKeyID, profile.Credential.KeyPair.ID),
		fmt.Sprintf("export %s=%s\n", location.AWSSecretAccessKey, profile.Credential.KeyPair.Secret),
		fmt.Sprintf("export %s=%s\n", ResticPassword, generatePassword()),
		fmt.Sprintf("export %s=s3:%s/%s\n", ResticRepository, s3Endpoint, repository),
		ResticCommand,
	}
}
