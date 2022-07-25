// Copyright 2021 The Kanister Authors.
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

package kopia

import (
	"bufio"
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kopia/kopia/repo"
	"github.com/kopia/kopia/repo/manifest"
	"github.com/kopia/kopia/repo/object"
	"github.com/kopia/kopia/snapshot"
	"github.com/kopia/kopia/snapshot/snapshotfs"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/format"
	kopiacmd "github.com/kanisterio/kanister/pkg/kopia/command"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
)

const (
	// defaultDataStoreGeneralContentCacheSizeMB is the default content cache size for general command workloads
	defaultDataStoreGeneralContentCacheSizeMB = 0

	// defaultDataStoreGeneralMetadataCacheSizeMB is the default metadata cache size for general command workloads
	defaultDataStoreGeneralMetadataCacheSizeMB = 500

	// TLSCertificateKey represents the key used to fetch the certificate
	// from the secret.
	TLSCertificateKey = "tls.crt"

	// BackupIdentifierKey is the artifact key used for kopia snapshot ID
	BackupIdentifierKey = "backupID"

	// ObjectStorePathOption is the option that specifies the repository to
	// use when describing repo
	ObjectStorePathOption = "objectStorePath"

	// DataStoreGeneralContentCacheSizeMBKey is the key to pass content cache size for general command workloads
	DataStoreGeneralContentCacheSizeMBKey = "dataStoreGeneralContentCacheSize"
	// DataStoreGeneralMetadataCacheSizeMBKey is the key to pass metadata cache size for general command workloads
	DataStoreGeneralMetadataCacheSizeMBKey = "dataStoreGeneralMetadataCacheSize"
)

// ExtractFingerprintFromCertSecret extracts the fingerprint from the given certificate secret
func ExtractFingerprintFromCertSecret(ctx context.Context, cli kubernetes.Interface, secretName, secretNamespace string) (string, error) {
	secret, err := cli.CoreV1().Secrets(secretNamespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrapf(err, "Failed to get Certificate Secret. Secret: %s", secretName)
	}

	certBytes, err := json.Marshal(secret.Data[TLSCertificateKey])
	if err != nil {
		return "", errors.Wrap(err, "Failed to marshal Certificate Secret Data")
	}

	var certString string
	if err := json.Unmarshal([]byte(certBytes), &certString); err != nil {
		return "", errors.Wrap(err, "Failed to unmarshal Certificate Secret Data")
	}

	decodedCertData, err := base64.StdEncoding.DecodeString(certString)
	if err != nil {
		return "", errors.Wrap(err, "Failed to decode Certificate Secret Data")
	}

	return extractFingerprintFromSliceOfBytes(decodedCertData)
}

// extractFingerprintFromSliceOfBytes extracts the fingeprint from the
// certificate data provided in slice of bytes (default type for secret.Data)
func extractFingerprintFromSliceOfBytes(pemData []byte) (string, error) {
	block, rest := pem.Decode([]byte(pemData))
	if block == nil || len(rest) > 0 {
		return "", errors.New("Failed to PEM Decode Kopia API Server Certificate Secret Data")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", errors.Wrap(err, "Failed to parse X509 Kopia API Server Certificate Secret Data")
	}

	fingerprint := sha256.Sum256(cert.Raw)
	return hex.EncodeToString(fingerprint[:]), nil
}

// ExtractFingerprintFromCertificateJSON fetch the fingerprint from a base64 encoded,
// certificate which is also type asserted into a string.
func ExtractFingerprintFromCertificateJSON(cert string) (string, error) {
	var certMap map[string]string

	if err := json.Unmarshal([]byte(cert), &certMap); err != nil {
		return "", errors.Wrap(err, "Failed to unmarshal Kopia API Server Certificate Secret Data")
	}

	decodedCertData, err := base64.StdEncoding.DecodeString(certMap[TLSCertificateKey])
	if err != nil {
		return "", errors.Wrap(err, "Failed to base64 decode Kopia API Server Certificate Secret Data")
	}

	fingerprint, err := extractFingerprintFromSliceOfBytes(decodedCertData)
	if err != nil {
		return "", errors.Wrap(err, "Failed to extract fingerprint Kopia API Server Certificate Secret Data")
	}

	return fingerprint, nil
}

// ExtractFingerprintFromCertificate fetch the fingerprint from a base64 encoded,
// certificate which is also type asserted into a string.
func ExtractFingerprintFromCertificate(cert string) (string, error) {
	fingerprint, err := extractFingerprintFromSliceOfBytes([]byte(cert))
	if err != nil {
		return "", errors.Wrap(err, "Failed to extract fingerprint Kopia API Server Certificate Secret Data")
	}

	return fingerprint, nil
}

// GetStreamingFileObjectIDFromSnapshot returns the kopia object ID of the fs.StreamingFile object from the repository
func GetStreamingFileObjectIDFromSnapshot(ctx context.Context, rep repo.Repository, path, backupID string) (object.ID, error) {
	// Example: if the path from the blueprint is `/mysql-backups/1/2/mysqldump.sql`, the given backupID
	// belongs to the root entry `/mysql-backups/1/2` with `mysqldump.sql` as a nested entry.
	// The goal here is to find the nested entry and extract the object ID

	// Load the kopia snapshot with the given backupID
	m, err := snapshot.LoadSnapshot(ctx, rep, manifest.ID(backupID))
	if err != nil {
		return "", errors.Wrapf(err, "Failed to load kopia snapshot with ID: %v", backupID)
	}

	// root entry of the kopia snapshot is a static directory with filepath.Dir(path) as its path
	if m.RootEntry == nil {
		return "", errors.New("No root entry found in kopia manifest")
	}
	rootEntry, err := snapshotfs.SnapshotRoot(rep, m)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to get root entry from kopia snapshot with ID: %v", backupID)
	}

	// Get the nested entry belonging to the backed up streaming file and return its object ID
	e, err := snapshotfs.GetNestedEntry(ctx, rootEntry, []string{filepath.Base(path)})
	if err != nil {
		return "", errors.Wrapf(err, "Failed to get nested entry from kopia snapshot: %v", filepath.Base(path))
	}

	return e.(object.HasObjectID).ObjectID(), nil
}

// GetDataStoreGeneralContentCacheSize finds and return content cache size from the options
func GetDataStoreGeneralContentCacheSize(opt map[string]int) int {
	if opt == nil {
		return defaultDataStoreGeneralContentCacheSizeMB
	}
	if contentCacheSize, ok := opt[DataStoreGeneralContentCacheSizeMBKey]; ok {
		return contentCacheSize
	}
	return defaultDataStoreGeneralContentCacheSizeMB
}

// GetDataStoreGeneralMetadataCacheSize finds and return metadata cache size from the options
func GetDataStoreGeneralMetadataCacheSize(opt map[string]int) int {
	if opt == nil {
		return defaultDataStoreGeneralMetadataCacheSizeMB
	}
	if metadataCacheSize, ok := opt[DataStoreGeneralContentCacheSizeMBKey]; ok {
		return metadataCacheSize
	}
	return defaultDataStoreGeneralMetadataCacheSizeMB
}

const (
	pathKey       = "path"
	typeKey       = "type"
	snapshotValue = "snapshot"
)

// SnapshotIDsFromSnapshot extracts root ID of a snapshot from the logs
func SnapshotIDsFromSnapshot(output string) (snapID, rootID string, err error) {
	if output == "" {
		return snapID, rootID, errors.New("Received empty output")
	}

	logs := regexp.MustCompile("[\r\n]").Split(output, -1)
	pattern := regexp.MustCompile(`Created snapshot with root ([^\s]+) and ID ([^\s]+).*$`)
	for _, l := range logs {
		// Log should contain "Created snapshot with root ABC and ID XYZ..."
		match := pattern.FindAllStringSubmatch(l, 1)
		if len(match) > 0 && len(match[0]) > 2 {
			snapID = match[0][2]
			rootID = match[0][1]
			return
		}
	}
	return snapID, rootID, errors.New("Failed to find Root ID from output")
}

// LatestSnapshotInfoFromManifestList returns snapshot ID and backup path of the latest snapshot from `manifests list` output
func LatestSnapshotInfoFromManifestList(output string) (string, string, error) {
	manifestList := []manifest.EntryMetadata{}
	snapID := ""
	backupPath := ""

	err := json.Unmarshal([]byte(output), &manifestList)
	if err != nil {
		return snapID, backupPath, errors.Wrap(err, "Failed to unmarshal manifest list")
	}
	for _, manifest := range manifestList {
		for key, value := range manifest.Labels {
			if key == pathKey {
				backupPath = value
			}
			if key == typeKey && value == snapshotValue {
				snapID = string(manifest.ID)
			}
		}
	}
	if snapID == "" {
		return "", "", errors.New("Failed to get latest snapshot ID from manifest list")
	}
	if backupPath == "" {
		return "", "", errors.New("Failed to get latest snapshot backup path from manifest list")
	}
	return snapID, backupPath, nil
}

// SnapshotInfoFromSnapshotCreateOutput returns snapshot ID and root ID from snapshot create output
func SnapshotInfoFromSnapshotCreateOutput(output string) (string, string, error) {
	snapID := ""
	rootID := ""
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		snapManifest := &snapshot.Manifest{}
		err := json.Unmarshal([]byte(scanner.Text()), snapManifest)
		if err != nil {
			continue
		}
		if snapManifest != nil {
			snapID = string(snapManifest.ID)
			if snapManifest.RootEntry != nil {
				rootID = string(snapManifest.RootEntry.ObjectID)
			}
		}
	}
	if snapID == "" {
		return "", "", errors.New("Failed to get snapshot ID from create snapshot output")
	}
	if rootID == "" {
		return "", "", errors.New("Failed to get root ID from create snapshot output")
	}
	return snapID, rootID, nil
}

// SnapSizeStatsFromSnapListAll returns a list of snapshot logical sizes assuming the input string
// is formatted as the output of a kopia snapshot list --all command.
func SnapSizeStatsFromSnapListAll(output string) (totalSizeB int64, numSnapshots int, err error) {
	if output == "" {
		return 0, 0, errors.New("Received empty output")
	}

	snapList, err := parseSnapshotManifestList(output)
	if err != nil {
		return 0, 0, errors.Wrap(err, "Parsing snapshot list output as snapshot manifest list")
	}

	totalSizeB = sumSnapshotSizes(snapList)

	return totalSizeB, len(snapList), nil
}

func sumSnapshotSizes(snapList []*snapshot.Manifest) (sum int64) {
	noSizeDataCount := 0
	for _, snapInfo := range snapList {
		if snapInfo.RootEntry == nil ||
			snapInfo.RootEntry.DirSummary == nil {
			noSizeDataCount++

			continue
		}

		sum += snapInfo.RootEntry.DirSummary.TotalFileSize
	}

	if noSizeDataCount > 0 {
		log.Error().Print("Found snapshot manifests without size data", field.M{"count": noSizeDataCount})
	}

	return sum
}

func parseSnapshotManifestList(output string) ([]*snapshot.Manifest, error) {
	snapInfoList := []*snapshot.Manifest{}

	if err := json.Unmarshal([]byte(output), &snapInfoList); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshal snapshot manifest list")
	}

	return snapInfoList, nil
}

// KopiaUserProfile is a duplicate of struct for Kopia user profiles since Profile struct is in internal/user package and could not be imported
type KopiaUserProfile struct {
	ManifestID manifest.ID `json:"-"`

	Username            string `json:"username"`
	PasswordHashVersion int    `json:"passwordHashVersion"`
	PasswordHash        []byte `json:"passwordHash"`
}

// GetMaintenanceOwnerForConnectedRepository executes maintenance info command, parses output
// and returns maintenance owner
func GetMaintenanceOwnerForConnectedRepository(
	cli kubernetes.Interface,
	namespace,
	pod,
	container,
	encryptionKey,
	configFilePath,
	logDirectory string,
) (string, error) {
	args := kopiacmd.MaintenanceInfoCommandArgs{
		CommandArgs: &kopiacmd.CommandArgs{
			EncryptionKey:  encryptionKey,
			ConfigFilePath: configFilePath,
			LogDirectory:   logDirectory,
		},
		GetJsonOutput: false,
	}
	cmd := kopiacmd.MaintenanceInfo(args)
	stdout, stderr, err := kube.Exec(cli, namespace, pod, container, cmd, nil)
	format.Log(pod, container, stdout)
	format.Log(pod, container, stderr)
	if err != nil {
		return "", err
	}
	parsedOwner := parseOutput(stdout)
	if parsedOwner == "" {
		return "", errors.New("Failed parsing maintenance info output to get owner")
	}
	return parsedOwner, nil
}

func parseOutput(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Owner") {
			arr := strings.Split(line, ":")
			if len(arr) == 2 {
				return strings.TrimSpace(arr[1])
			}
		}
	}
	return ""
}
