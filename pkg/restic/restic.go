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
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/aws"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/location"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/secrets"
)

const (
	PasswordIncorrect = "Password is incorrect"
	RepoDoesNotExist  = "Repo does not exist"
)

func shCommand(command string) []string {
	return []string{"bash", "-o", "errexit", "-o", "pipefail", "-c", command}
}

// BackupCommandByTag returns restic backup command with tag
func BackupCommandByTag(profile *param.Profile, repository, backupTag, includePath, encryptionKey string, insecureTLS bool) ([]string, error) {
	cmd, err := resticArgs(profile, repository, encryptionKey)
	if err != nil {
		return nil, err
	}
	cmd = append(cmd, "backup", "--tag", backupTag, includePath)
	if insecureTLS {
		cmd = append(cmd, "--insecure-tls")
	}
	command := strings.Join(cmd, " ")
	return shCommand(command), nil
}

// RestoreCommandByID returns restic restore command with snapshotID as the identifier
func RestoreCommandByID(profile *param.Profile, repository, id, restorePath, encryptionKey string, insecureTLS bool) ([]string, error) {
	cmd, err := resticArgs(profile, repository, encryptionKey)
	if err != nil {
		return nil, err
	}
	cmd = append(cmd, "restore", id, "--target", restorePath)
	if insecureTLS {
		cmd = append(cmd, "--insecure-tls")
	}
	command := strings.Join(cmd, " ")
	return shCommand(command), nil
}

// RestoreCommandByTag returns restic restore command with tag as the identifier
func RestoreCommandByTag(profile *param.Profile, repository, tag, restorePath, encryptionKey string, insecureTLS bool) ([]string, error) {
	cmd, err := resticArgs(profile, repository, encryptionKey)
	if err != nil {
		return nil, err
	}
	cmd = append(cmd, "restore", "--tag", tag, "latest", "--target", restorePath)
	if insecureTLS {
		cmd = append(cmd, "--insecure-tls")
	}
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

// LatestSnapshotsCommand returns restic snapshots command for last snapshots
func LatestSnapshotsCommand(profile *param.Profile, repository, encryptionKey string, insecureTLS bool) ([]string, error) {
	cmd, err := resticArgs(profile, repository, encryptionKey)
	if err != nil {
		return nil, err
	}
	cmd = append(cmd, "snapshots", "--last", "--json")
	if insecureTLS {
		cmd = append(cmd, "--insecure-tls")
	}
	command := strings.Join(cmd, " ")
	return shCommand(command), nil
}

// SnapshotsCommandByTag returns restic snapshots command
func SnapshotsCommandByTag(profile *param.Profile, repository, tag, encryptionKey string, insecureTLS bool) ([]string, error) {
	cmd, err := resticArgs(profile, repository, encryptionKey)
	if err != nil {
		return nil, err
	}
	cmd = append(cmd, "snapshots", "--tag", tag, "--json")
	if insecureTLS {
		cmd = append(cmd, "--insecure-tls")
	}
	command := strings.Join(cmd, " ")
	return shCommand(command), nil
}

// InitCommand returns restic init command
func InitCommand(profile *param.Profile, repository, encryptionKey string, insecureTLS bool) ([]string, error) {
	cmd, err := resticArgs(profile, repository, encryptionKey)
	if err != nil {
		return nil, err
	}
	cmd = append(cmd, "init")
	if insecureTLS {
		cmd = append(cmd, "--insecure-tls")
	}
	command := strings.Join(cmd, " ")
	return shCommand(command), nil
}

// ForgetCommandByID returns restic forget command
func ForgetCommandByID(profile *param.Profile, repository, id, encryptionKey string, insecureTLS bool) ([]string, error) {
	cmd, err := resticArgs(profile, repository, encryptionKey)
	if err != nil {
		return nil, err
	}
	cmd = append(cmd, "forget", id)
	if insecureTLS {
		cmd = append(cmd, "--insecure-tls")
	}
	command := strings.Join(cmd, " ")
	return shCommand(command), nil
}

// PruneCommand returns restic prune command
func PruneCommand(profile *param.Profile, repository, encryptionKey string, insecureTLS bool) ([]string, error) {
	cmd, err := resticArgs(profile, repository, encryptionKey)
	if err != nil {
		return nil, err
	}
	cmd = append(cmd, "prune")
	if insecureTLS {
		cmd = append(cmd, "--insecure-tls")
	}
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
		cmd, err = resticAzureArgs(profile, repository)
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

func resticS3CredentialSecretArgs(secret *corev1.Secret) ([]string, error) {
	creds, err := secrets.ExtractAWSCredentials(context.Background(), secret, aws.AssumeRoleDurationDefault)
	if err != nil {
		return nil, err
	}
	args := []string{
		fmt.Sprintf("export %s=%s\n", location.AWSAccessKeyID, creds.AccessKeyID),
		fmt.Sprintf("export %s=%s\n", location.AWSSecretAccessKey, creds.SecretAccessKey),
	}
	if creds.SessionToken != "" {
		args = append(args, fmt.Sprintf("export %s=%s\n", location.AWSSessionToken, creds.SessionToken))
	}
	return args, nil
}

func resticGCSArgs(profile *param.Profile, repository string) []string {
	return []string{
		fmt.Sprintf("export %s=%s\n", location.GoogleProjectID, profile.Credential.KeyPair.ID),
		fmt.Sprintf("export %s=%s\n", location.GoogleCloudCreds, consts.GoogleCloudCredsFilePath),
		fmt.Sprintf("export %s=gs:%s/\n", ResticRepository, strings.Replace(repository, "/", ":/", 1)),
	}
}

func resticAzureArgs(profile *param.Profile, repository string) ([]string, error) {
	var storageAccountID, storageAccountKey string
	switch profile.Credential.Type {
	case param.CredentialTypeKeyPair:
		storageAccountID = profile.Credential.KeyPair.ID
		storageAccountKey = profile.Credential.KeyPair.Secret
	case param.CredentialTypeSecret:
		creds, err := secrets.ExtractAzureCredentials(profile.Credential.Secret)
		if err != nil {
			return nil, err
		}
		storageAccountID = creds.StorageAccount
		storageAccountKey = creds.StorageKey
	}

	return []string{
		fmt.Sprintf("export %s=%s\n", location.AzureStorageAccount, storageAccountID),
		fmt.Sprintf("export %s=%s\n", location.AzureStorageKey, storageAccountKey),
		fmt.Sprintf("export %s=azure:%s/\n", ResticRepository, strings.Replace(repository, "/", ":/", 1)),
	}, nil
}

// GetOrCreateRepository will check if the repository already exists and initialize one if not
func GetOrCreateRepository(
	ctx context.Context,
	cli kubernetes.Interface,
	namespace,
	pod,
	container,
	artifactPrefix,
	encryptionKey string,
	insecureTLS bool,
	profile *param.Profile,
) error {
	_, _, err := getLatestSnapshots(ctx, profile, artifactPrefix, encryptionKey, insecureTLS, cli, namespace, pod, container)
	if err == nil {
		return nil
	}
	// Create a repository
	cmd, err := InitCommand(profile, artifactPrefix, encryptionKey, insecureTLS)
	if err != nil {
		return errors.Wrap(err, "Failed to create init command")
	}
	stdout, stderr, err := kube.Exec(ctx, cli, namespace, pod, container, cmd, nil)
	format.Log(pod, container, stdout)
	format.Log(pod, container, stderr)
	return errors.Wrapf(err, "Failed to create object store backup location")
}

// CheckIfRepoIsReachable checks if repo can be reached by trying to list snapshots
func CheckIfRepoIsReachable(
	ctx context.Context,
	profile *param.Profile,
	artifactPrefix string,
	encryptionKey string,
	insecureTLS bool,
	cli kubernetes.Interface,
	namespace string,
	pod string,
	container string,
) error {
	_, stderr, err := getLatestSnapshots(ctx, profile, artifactPrefix, encryptionKey, insecureTLS, cli, namespace, pod, container)
	if IsPasswordIncorrect(stderr) { // If password didn't work
		return errors.New(PasswordIncorrect)
	}
	if DoesRepoExist(stderr) {
		return errors.New(RepoDoesNotExist)
	}
	if err != nil {
		return errors.Wrap(err, "Failed to list snapshots")
	}
	return nil
}

//nolint:unparam
func getLatestSnapshots(
	ctx context.Context,
	profile *param.Profile,
	artifactPrefix string,
	encryptionKey string,
	insecureTLS bool,
	cli kubernetes.Interface,
	namespace string,
	pod string,
	container string,
) (string, string, error) {
	// Use the latest snapshots command to check if the repository exists
	cmd, err := LatestSnapshotsCommand(profile, artifactPrefix, encryptionKey, insecureTLS)
	if err != nil {
		return "", "", errors.Wrap(err, "Failed to create snapshot command")
	}
	stdout, stderr, err := kube.Exec(ctx, cli, namespace, pod, container, cmd, nil)
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
	snapID := result[0]["short_id"]
	return snapID.(string), nil
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
func SnapshotStatsFromBackupLog(output string) (fileCount string, backupSize string, phySize string) {
	if output == "" {
		return "", "", ""
	}
	logs := regexp.MustCompile("[\n]").Split(output, -1)
	// Log should contain "processed %d files, %.3f [Xi]B in mm:ss"
	logicalPattern := regexp.MustCompile(`processed\s([\d]+)\sfiles,\s([\d]+(\.[\d]+)?\s([TGMK]i)?B)\sin\s`)
	// Log should contain "Added to the repo: %.3f [Xi]B"
	physicalPattern := regexp.MustCompile(`^Added to the repo: ([\d]+(\.[\d]+)?\s([TGMK]i)?B)$`)

	for _, l := range logs {
		logMatch := logicalPattern.FindAllStringSubmatch(l, 1)
		phyMatch := physicalPattern.FindAllStringSubmatch(l, 1)
		switch {
		case logMatch != nil:
			if len(logMatch) >= 1 && len(logMatch[0]) >= 3 {
				// Expect in order:
				// 0: entire match,
				// 1: first submatch == file count,
				// 2: second submatch == size string
				fileCount = logMatch[0][1]
				backupSize = logMatch[0][2]
			}
		case phyMatch != nil:
			if len(phyMatch) >= 1 && len(phyMatch[0]) >= 2 {
				// Expect in order:
				// 0: entire match,
				// 1: first submatch == size string,
				phySize = phyMatch[0][1]
			}
		}
	}
	return fileCount, backupSize, phySize
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
	pattern := regexp.MustCompile(`Stats in\s+(.*?)\s+mode:`)
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

// DoesRepoExist checks if repo exists from Snapshot Command log
func DoesRepoExist(output string) bool {
	return strings.Contains(output, "Is there a repository at the following location?")
}

// SpaceFreedFromPruneLog gets the space freed from the prune log output
// For reference, here is the logging command from restic codebase:
//
//		Verbosef("will delete %d packs and rewrite %d packs, this frees %s\n",
//	              len(removePacks), len(rewritePacks), formatBytes(uint64(removeBytes)))
func SpaceFreedFromPruneLog(output string) string {
	var spaceFreed string
	logs := regexp.MustCompile("[\n]").Split(output, -1)
	// Log should contain "will delete x packs and rewrite y packs, this frees zz.zzz [[TGMK]i]B"
	pattern := regexp.MustCompile(`^will delete \d+ packs and rewrite \d+ packs, this frees ([\d]+(\.[\d]+)?\s([TGMK]i)?B)$`)
	for _, l := range logs {
		match := pattern.FindAllStringSubmatch(l, 1)
		if len(match) > 0 && len(match[0]) > 1 {
			spaceFreed = match[0][1]
		}
	}
	return spaceFreed
}

// ParseResticSizeStringBytes parses size strings as formatted by restic to
// a int64 number of bytes
func ParseResticSizeStringBytes(sizeStr string) int64 {
	components := regexp.MustCompile(`[\s]`).Split(sizeStr, -1)
	if len(components) != 2 {
		return 0
	}
	sizeNumStr := components[0]
	sizeNum, err := strconv.ParseFloat(sizeNumStr, 64)
	if err != nil {
		return 0
	}
	if sizeNum < 0 {
		return 0
	}
	magnitudeStr := components[1]
	pattern := regexp.MustCompile(`^(([TGMK]i)?B)$`)
	match := pattern.FindAllStringSubmatch(magnitudeStr, 1)
	if match != nil {
		if len(match) != 1 || len(match[0]) != 3 {
			return 0
		}
		magnitude := match[0][1]
		switch magnitude {
		case "TiB":
			return int64(sizeNum * (1 << 40))
		case "GiB":
			return int64(sizeNum * (1 << 30))
		case "MiB":
			return int64(sizeNum * (1 << 20))
		case "KiB":
			return int64(sizeNum * (1 << 10))
		case "B":
			return int64(sizeNum)
		default:
			return 0
		}
	}
	return 0
}
