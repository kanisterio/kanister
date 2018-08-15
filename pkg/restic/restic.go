package restic

import (
	"fmt"
	"strings"

	"github.com/kanisterio/kanister/pkg/location"
	"github.com/kanisterio/kanister/pkg/param"
)

// BackupCommand returns restic backup command
func BackupCommand(profile *param.Profile, repository string) (string, error) {
	cmd, err := resticArgs(profile, repository)
	if err != nil {
		return "", err
	}
	cmd = append(cmd, "backup")
	command := strings.Join(cmd, " ")
	return command, nil
}

// RestoreCommand returns restic restore command
func RestoreCommand(profile *param.Profile, repository string) (string, error) {
	cmd, err := resticArgs(profile, repository)
	if err != nil {
		return "", err
	}
	cmd = append(cmd, "restore")
	command := strings.Join(cmd, " ")
	return command, nil
}

// SnapshotsCommand returns restic snapshots command
func SnapshotsCommand(profile *param.Profile, repository string) (string, error) {
	cmd, err := resticArgs(profile, repository)
	if err != nil {
		return "", err
	}
	cmd = append(cmd, "snapshots")
	command := strings.Join(cmd, " ")
	return command, nil
}

// InitCommand returns restic init command
func InitCommand(profile *param.Profile, repository string) (string, error) {
	cmd, err := resticArgs(profile, repository)
	if err != nil {
		return "", err
	}
	cmd = append(cmd, "init")
	command := strings.Join(cmd, " ")
	return command, nil
}

// ForgetCommand returns restic forget command
func ForgetCommand(profile *param.Profile, repository string) (string, error) {
	cmd, err := resticArgs(profile, repository)
	if err != nil {
		return "", err
	}
	cmd = append(cmd, "forget")
	command := strings.Join(cmd, " ")
	return command, nil
}

// PruneCommand returns restic prune command
func PruneCommand(profile *param.Profile, repository string) (string, error) {
	cmd, err := resticArgs(profile, repository)
	if err != nil {
		return "", err
	}
	cmd = append(cmd, "prune")
	command := strings.Join(cmd, " ")
	return command, nil
}

const (
	ResticPassword   = "RESTIC_PASSWORD"
	ResticRepository = "RESTIC_REPOSITORY"
	ResticCommand    = "restic"
)

func resticArgs(profile *param.Profile, repository string) ([]string, error) {
	pwd, err := generatePassword()
	if err != nil {
		return nil, err
	}
	return []string{
		fmt.Sprintf("export %s=%s\n", location.AWSAccessKeyID, profile.Credential.KeyPair.ID),
		fmt.Sprintf("export %s=%s\n", location.AWSSecretAccessKey, profile.Credential.KeyPair.Secret),
		fmt.Sprintf("export %s=%s\n", ResticPassword, pwd),
		fmt.Sprintf("export %s=s3:s3.amazonaws.com/%s\n", ResticRepository, repository),
		ResticCommand,
	}, nil
}
