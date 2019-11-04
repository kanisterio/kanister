// Copyright 2019 The Kanister Authors.
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

package restic

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/location"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/secrets"
)

const (
	GoogleCloudCredsFilePath = "/tmp/creds.txt"
	PasswordIncorrect        = "Password is incorrect"
	RepoDoesNotExist         = "Repo does not exist"
)

func shCommand(command string) []string {
	return []string{"bash", "-o", "errexit", "-o", "pipefail", "-c", command}
}

// BackupCommandByID returns restic backup command
func BackupCommandByID(profile *param.Profile, repository, pathToBackup, encryptionKey string) ([]string, error) {
	cmd, err := resticArgs(profile, repository, encryptionKey)
	if err != nil {
		return nil, err
	}
	cmd = append(cmd, "backup", pathToBackup)
	command := strings.Join(cmd, " ")
	return shCommand(command), nil
}

// BackupCommandByTag returns restic backup command with tag
func BackupCommandByTag(profile *param.Profile, repository, backupTag, includePath, encryptionKey string) ([]string, error) {
	cmd, err := resticArgs(profile, repository, encryptionKey)
	if err != nil {
		return nil, err
	}
	cmd = append(cmd, "backup", "--tag", backupTag, includePath)
	command := strings.Join(cmd, " ")
	return shCommand(command), nil
}

// RestoreCommandByID returns restic restore command with snapshotID as the identifier
func RestoreCommandByID(profile *param.Profile, repository, id, restorePath, encryptionKey string) ([]string, error) {
	cmd, err := resticArgs(profile, repository, encryptionKey)
	if err != nil {
		return nil, err
	}
	cmd = append(cmd, "restore", id, "--target", restorePath)
	command := strings.Join(cmd, " ")
	return shCommand(command), nil
}

// RestoreCommandByTag returns restic restore command with tag as the identifier
func RestoreCommandByTag(profile *param.Profile, repository, tag, restorePath, encryptionKey string) ([]string, error) {
	cmd, err := resticArgs(profile, repository, encryptionKey)
	if err != nil {
		return nil, err
	}
	cmd = append(cmd, "restore", "--tag", tag, "latest", "--target", restorePath)
	command := strings.Join(cmd, " ")
	return shCommand(command), nil
}

// SnapshotsCommand returns restic snapshots command
func SnapshotsCommand(profile *param.Profile, repository, encryptionKey string) ([]string, error) {
	cmd, err := resticArgs(profile, repository, encryptionKey)
	if err != nil {
		return nil, err
	}
	cmd = append(cmd, "snapshots", "--json")
	command := strings.Join(cmd, " ")
	return shCommand(command), nil
}

// SnapshotsCommandByTag returns restic snapshots command
func SnapshotsCommandByTag(profile *param.Profile, repository, tag, encryptionKey string) ([]string, error) {
	cmd, err := resticArgs(profile, repository, encryptionKey)
	if err != nil {
		return nil, err
	}
	cmd = append(cmd, "snapshots", "--tag", tag, "--json")
	command := strings.Join(cmd, " ")
	return shCommand(command), nil
}

// InitCommand returns restic init command
func InitCommand(profile *param.Profile, repository, encryptionKey string) ([]string, error) {
	cmd, err := resticArgs(profile, repository, encryptionKey)
	if err != nil {
		return nil, err
	}
	cmd = append(cmd, "init")
	command := strings.Join(cmd, " ")
	return shCommand(command), nil
}

// ForgetCommandByTag returns restic forget command
func ForgetCommandByTag(profile *param.Profile, repository, tag, encryptionKey string) ([]string, error) {
	cmd, err := resticArgs(profile, repository, encryptionKey)
	if err != nil {
		return nil, err
	}
	cmd = append(cmd, "forget", "--tag", tag)
	command := strings.Join(cmd, " ")
	return shCommand(command), nil
}

// ForgetCommandByID returns restic forget command
func ForgetCommandByID(profile *param.Profile, repository, id, encryptionKey string) ([]string, error) {
	cmd, err := resticArgs(profile, repository, encryptionKey)
	if err != nil {
		return nil, err
	}
	cmd = append(cmd, "forget", id)
	command := strings.Join(cmd, " ")
	return shCommand(command), nil
}

// PruneCommand returns restic prune command
func PruneCommand(profile *param.Profile, repository, encryptionKey string) ([]string, error) {
	cmd, err := resticArgs(profile, repository, encryptionKey)
	if err != nil {
		return nil, err
	}
	cmd = append(cmd, "prune")
	command := strings.Join(cmd, " ")
	return shCommand(command), nil
}

// StatsCommandByID returns restic stats command
func StatsCommandByID(profile *param.Profile, repository, id, mode, encryptionKey string) ([]string, error) {
	cmd, err := resticArgs(profile, repository, encryptionKey)
	if err != nil {
		return nil, err
	}
	cmd = append(cmd, "stats", id, "--mode", mode)
	command := strings.Join(cmd, " ")
	return shCommand(command), nil
}

const (
	ResticPassword   = "RESTIC_PASSWORD"
	ResticRepository = "RESTIC_REPOSITORY"
	ResticCommand    = "restic"
	awsS3Endpoint    = "s3.amazonaws.com"
)

func resticArgs(profile *param.Profile, repository, encryptionKey string) ([]string, error) {
	var cmd []string
	var err error
	switch profile.Location.Type {
	case crv1alpha1.LocationTypeS3Compliant:
		cmd, err = resticS3Args(profile, repository)
	case crv1alpha1.LocationTypeGCS:
		cmd = resticGCSArgs(profile, repository)
	case crv1alpha1.LocationTypeAzure:
		cmd = resticAzureArgs(profile, repository)
	default:
		return nil, errors.New("Unsupported type '%s' for the location")
	}
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get arguments")
	}
	return append(cmd, fmt.Sprintf("export %s=%s\n", ResticPassword, encryptionKey), ResticCommand), nil
}

func resticS3Args(profile *param.Profile, repository string) ([]string, error) {
	s3Endpoint := awsS3Endpoint
	if profile.Location.Endpoint != "" {
		s3Endpoint = profile.Location.Endpoint
	}
	if strings.HasSuffix(s3Endpoint, "/") {
		log.Debug().Print("Removing trailing slashes from the endpoint")
		s3Endpoint = strings.TrimRight(s3Endpoint, "/")
	}
	args, err := resticS3CredentialArgs(profile.Credential)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create args from credential")
	}
	args = append(args, fmt.Sprintf("export %s=s3:%s/%s\n", ResticRepository, s3Endpoint, repository))
	return args, nil
}

func resticS3CredentialArgs(creds param.Credential) ([]string, error) {
	switch creds.Type {
	case param.CredentialTypeKeyPair:
		return []string{
			fmt.Sprintf("export %s=%s\n", location.AWSAccessKeyID, creds.KeyPair.ID),
			fmt.Sprintf("export %s=%s\n", location.AWSSecretAccessKey, creds.KeyPair.Secret),
		}, nil
	case param.CredentialTypeSecret:
		return resticS3CredentialSecretArgs(creds.Secret)
	default:
		return nil, errors.Errorf("Unsupported type '%s' for credentials", creds.Type)
	}
}

func resticS3CredentialSecretArgs(secret *v1.Secret) ([]string, error) {
	if err := secrets.ValidateAWSCredentials(secret); err != nil {
		return nil, err
	}
	args := []string{
		fmt.Sprintf("export %s=%s\n", location.AWSAccessKeyID, secret.Data[secrets.AWSAccessKeyID]),
		fmt.Sprintf("export %s=%s\n", location.AWSSecretAccessKey, secret.Data[secrets.AWSSecretAccessKey]),
	}
	if _, ok := secret.Data[secrets.AWSSessionToken]; ok {
		args = append(args, fmt.Sprintf("export %s=%s\n", location.AWSSessionToken, secret.Data[secrets.AWSSessionToken]))
	}
	return args, nil
}

func resticGCSArgs(profile *param.Profile, repository string) []string {
	return []string{
		fmt.Sprintf("export %s=%s\n", location.GoogleProjectId, profile.Credential.KeyPair.ID),
		fmt.Sprintf("export %s=%s\n", location.GoogleCloudCreds, GoogleCloudCredsFilePath),
		fmt.Sprintf("export %s=gs:%s/\n", ResticRepository, strings.Replace(repository, "/", ":/", 1)),
	}
}

func resticAzureArgs(profile *param.Profile, repository string) []string {
	return []string{
		fmt.Sprintf("export %s=%s\n", location.AzureStorageAccount, profile.Credential.KeyPair.ID),
		fmt.Sprintf("export %s=%s\n", location.AzureStorageKey, profile.Credential.KeyPair.Secret),
		fmt.Sprintf("export %s=azure:%s/\n", ResticRepository, strings.Replace(repository, "/", ":/", 1)),
	}
}

// GetOrCreateRepository will check if the repository already exists and initialize one if not
func GetOrCreateRepository(cli kubernetes.Interface, namespace, pod, container, artifactPrefix, encryptionKey string, profile *param.Profile) error {
	stdout, stderr, err := listSnapshots(profile, artifactPrefix, encryptionKey, cli, namespace, pod, container)
	format.Log(pod, container, stdout)
	format.Log(pod, container, stderr)
	if err == nil {
		return nil
	}
	// Create a repository
	cmd, err := InitCommand(profile, artifactPrefix, encryptionKey)
	if err != nil {
		return errors.Wrap(err, "Failed to create init command")
	}
	stdout, stderr, err = kube.Exec(cli, namespace, pod, container, cmd, nil)
	format.Log(pod, container, stdout)
	format.Log(pod, container, stderr)
	return errors.Wrapf(err, "Failed to create object store backup location")
}

// GetSnapshotIDs checks if repo is reachable with current encryptionKey, and get a list of snapshot IDs
func GetSnapshotIDs(profile *param.Profile, cli kubernetes.Interface, artifactPrefix, encryptionKey, namespace, pod, container string) ([]string, error) {
	stdout, err := CheckIfRepoIsReachable(profile, artifactPrefix, encryptionKey, cli, namespace, pod, container)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to connect to object store location")
	}
	// parse snapshots for list of IDs
	snapshots, err := SnapshotIDsFromSnapshotCommand(stdout)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to list snapshots")
	}
	return snapshots, nil
}

// CheckIfRepoIsReachable checks if repo can be reached by trying to list snapshots
func CheckIfRepoIsReachable(profile *param.Profile, artifactPrefix string, encryptionKey string, cli kubernetes.Interface, namespace string, pod string, container string) (string, error) {
	stdout, stderr, err := listSnapshots(profile, artifactPrefix, encryptionKey, cli, namespace, pod, container)
	if IsPasswordIncorrect(stderr) { // If password didn't work
		return "", errors.New(PasswordIncorrect)
	}
	if DoesRepoExist(stderr) {
		return "", errors.New(RepoDoesNotExist)
	}
	if err != nil {
		return "", errors.Wrap(err, "Failed to list snapshots")
	}
	return stdout, nil
}

func listSnapshots(profile *param.Profile, artifactPrefix string, encryptionKey string, cli kubernetes.Interface, namespace string, pod string, container string) (string, string, error) {
	// Use the snapshots command to check if the repository exists
	cmd, err := SnapshotsCommand(profile, artifactPrefix, encryptionKey)
	if err != nil {
		return "", "", errors.Wrap(err, "Failed to create snapshot command")
	}
	stdout, stderr, err := kube.Exec(cli, namespace, pod, container, cmd, nil)
	format.Log(pod, container, stdout)
	format.Log(pod, container, stderr)
	return stdout, stderr, err
}

// SnapshotIDFromSnapshotLog gets the SnapshotID from Snapshot Command log
func SnapshotIDFromSnapshotLog(output string) (string, error) {
	var result []map[string]interface{}
	err := json.Unmarshal([]byte(output), &result)
	if err != nil {
		return "", errors.WithMessage(err, "Failed to unmarshall output from snapshotCommand")
	}
	if len(result) == 0 {
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
	// Log should contain "snapshot ABC123 saved"
	pattern := regexp.MustCompile(`snapshot\s(.*?)\ssaved$`)
	for _, l := range logs {
		match := pattern.FindAllStringSubmatch(l, 1)
		if match != nil {
			if len(match) >= 1 && len(match[0]) >= 2 {
				return match[0][1]
			}
		}
	}
	return ""
}

// SnapshotStatsFromBackupLog gets the Snapshot file count and size from Backup Command log
func SnapshotStatsFromBackupLog(output string) (fileCount string, backupSize string) {
	if output == "" {
		return "", ""
	}
	logs := regexp.MustCompile("[\n]").Split(output, -1)
	// Log should contain "processed %d files, %.3f [Xi]B in mm:ss"
	pattern := regexp.MustCompile(`processed\s([\d]+)\sfiles,\s([\d]+(\.[\d]+)?\s([TGMK]i)?B)\sin\s`)
	for _, l := range logs {
		match := pattern.FindAllStringSubmatch(l, 1)
		if match != nil {
			if len(match) >= 1 && len(match[0]) >= 3 {
				// Expect in order:
				// 0: entire match,
				// 1: first submatch == file count,
				// 2: second submatch == size string
				return match[0][1], match[0][2]
			}
		}
	}
	return "", ""
}

// SnapshotStatsFromStatsLog gets the Snapshot Stats from Stats Command log
func SnapshotStatsFromStatsLog(output string) (string, string, string) {
	mode := SnapshotStatsModeFromStatsLog(output)
	if output == "" {
		return "", "", ""
	}
	var fileCount string
	var size string
	logs := regexp.MustCompile("[\n]").Split(output, -1)
	var pattern1 *regexp.Regexp
	// Log should contain "Total File Count:   xx"
	pattern1 = regexp.MustCompile(`Total File Count:\s+(.*?)$`)
	if mode == "raw-data" {
		// Log should contain "Total Blob Count:   xx"
		pattern1 = regexp.MustCompile(`Total Blob Count:\s+(.*?)$`)
	}
	// Log should contain "Total Size:   xx"
	pattern2 := regexp.MustCompile(`Total Size: \s+(.*?)$`)
	for _, l := range logs {
		match1 := pattern1.FindAllStringSubmatch(l, 1)
		if len(match1) > 0 && len(match1[0]) > 1 {
			fileCount = match1[0][1]
		}
		match2 := pattern2.FindAllStringSubmatch(l, 1)
		if len(match2) > 0 && len(match2[0]) > 1 {
			size = match2[0][1]
		}
	}
	return mode, fileCount, size
}

// SnapshotStatsModeFromStatsLog gets the Stats mode from Stats Command log
func SnapshotStatsModeFromStatsLog(output string) string {
	logs := regexp.MustCompile("[\n]").Split(output, -1)
	// Log should contain "Stats for .... in  xx mode"
	pattern := regexp.MustCompile(`Stats for.*in\s+(.*?)\s+mode:`)
	for _, l := range logs {
		match := pattern.FindAllStringSubmatch(l, 1)
		if len(match) > 0 && len(match[0]) > 1 {
			return match[0][1]
		}
	}
	return ""
}

// IsPasswordIncorrect checks if password was wrong from Snapshot Command log
func IsPasswordIncorrect(output string) bool {
	return strings.Contains(output, "wrong password")
}

// DoesRepoExists checks if repo exists from Snapshot Command log
func DoesRepoExist(output string) bool {
	return strings.Contains(output, "Is there a repository at the following location?")
}

// SnapshotIDFromSnapshotLog gets the SnapshotID from Snapshot Command log
func SnapshotIDsFromSnapshotCommand(output string) ([]string, error) {
	var snapIds []string
	var result []map[string]interface{}
	err := json.Unmarshal([]byte(output), &result)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to unmarshall output from snapshotCommand")
	}
	if len(result) == 0 {
		return nil, errors.New("Snapshots not found")
	}
	for _, r := range result {
		if r["short_id"] != nil {
			snapIds = append(snapIds, r["short_id"].(string))
		}
	}
	return snapIds, nil
}

// SpaceFreedFromPruneLog gets the space freed from the prune log output
// Reference logging commad from restic codebase:
// Verbosef("will delete %d packs and rewrite %d packs, this frees %s\n",
//		len(removePacks), len(rewritePacks), formatBytes(uint64(removeBytes)))
func SpaceFreedFromPruneLog(output string) string {
	var spaceFreed string
	logs := regexp.MustCompile("[\n]").Split(output, -1)
	// Log should contain "will delete x packs and rewrite y packs, this frees zz.zzz [[GMK]i]B"
	pattern := regexp.MustCompile(`^will delete \d+ packs and rewrite \d+ packs, this frees ([\d]+(\.[\d]+)?\s([TGMK]i)?B)$`)
	for _, l := range logs {
		match := pattern.FindAllStringSubmatch(l, 1)
		if len(match) > 0 && len(match[0]) > 1 {
			spaceFreed = match[0][1]
		}
	}
	return spaceFreed
}
