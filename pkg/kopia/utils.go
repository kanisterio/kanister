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
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"path/filepath"
	"strings"

	"github.com/kanisterio/errkit"
	"github.com/kopia/kopia/repo"
	"github.com/kopia/kopia/repo/manifest"
	"github.com/kopia/kopia/repo/object"
	"github.com/kopia/kopia/snapshot"
	"github.com/kopia/kopia/snapshot/snapshotfs"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kopiacmd "github.com/kanisterio/kanister/pkg/kopia/command"
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
	// ServerUsernameFormat is used to construct server username for Kopia Repository Server Status Command
	ServerUsernameFormat = "%s@%s"
	// KanisterAdminUsername is the username for the user with Admin privileges
	KanisterAdminUsername = "kanister-admin"
	defaultServerHostname = "data-mover-server-pod"
)

// ExtractFingerprintFromCertSecret extracts the fingerprint from the given certificate secret
func ExtractFingerprintFromCertSecret(ctx context.Context, cli kubernetes.Interface, secretName, secretNamespace string) (string, error) {
	secret, err := cli.CoreV1().Secrets(secretNamespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return "", errkit.Wrap(err, "Failed to get Certificate Secret.", "secretName", secretName)
	}

	certBytes, err := json.Marshal(secret.Data[TLSCertificateKey])
	if err != nil {
		return "", errkit.Wrap(err, "Failed to marshal Certificate Secret Data")
	}

	var certString string
	if err := json.Unmarshal([]byte(certBytes), &certString); err != nil {
		return "", errkit.Wrap(err, "Failed to unmarshal Certificate Secret Data")
	}

	decodedCertData, err := base64.StdEncoding.DecodeString(certString)
	if err != nil {
		return "", errkit.Wrap(err, "Failed to decode Certificate Secret Data")
	}

	return extractFingerprintFromSliceOfBytes(decodedCertData)
}

// extractFingerprintFromSliceOfBytes extracts the fingeprint from the
// certificate data provided in slice of bytes (default type for secret.Data)
func extractFingerprintFromSliceOfBytes(pemData []byte) (string, error) {
	block, rest := pem.Decode([]byte(pemData))
	if block == nil || len(rest) > 0 {
		return "", errkit.New("Failed to PEM Decode Kopia API Server Certificate Secret Data")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", errkit.Wrap(err, "Failed to parse X509 Kopia API Server Certificate Secret Data")
	}

	fingerprint := sha256.Sum256(cert.Raw)
	return hex.EncodeToString(fingerprint[:]), nil
}

// ExtractFingerprintFromCertificateJSON fetch the fingerprint from a base64 encoded,
// certificate which is also type asserted into a string.
func ExtractFingerprintFromCertificateJSON(cert string) (string, error) {
	var certMap map[string]string

	if err := json.Unmarshal([]byte(cert), &certMap); err != nil {
		return "", errkit.Wrap(err, "Failed to unmarshal Kopia API Server Certificate Secret Data")
	}

	decodedCertData, err := base64.StdEncoding.DecodeString(certMap[TLSCertificateKey])
	if err != nil {
		return "", errkit.Wrap(err, "Failed to base64 decode Kopia API Server Certificate Secret Data")
	}

	fingerprint, err := extractFingerprintFromSliceOfBytes(decodedCertData)
	if err != nil {
		return "", errkit.Wrap(err, "Failed to extract fingerprint Kopia API Server Certificate Secret Data")
	}

	return fingerprint, nil
}

// ExtractFingerprintFromCertificate fetch the fingerprint from a base64 encoded,
// certificate which is also type asserted into a string.
func ExtractFingerprintFromCertificate(cert string) (string, error) {
	fingerprint, err := extractFingerprintFromSliceOfBytes([]byte(cert))
	if err != nil {
		return "", errkit.Wrap(err, "Failed to extract fingerprint Kopia API Server Certificate Secret Data")
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
		return object.ID{}, errkit.Wrap(err, "Failed to load kopia snapshot with ID", "backupId", backupID)
	}

	// root entry of the kopia snapshot is a static directory with filepath.Dir(path) as its path
	if m.RootEntry == nil {
		return object.ID{}, errkit.New("No root entry found in kopia manifest")
	}
	rootEntry, err := snapshotfs.SnapshotRoot(rep, m)
	if err != nil {
		return object.ID{}, errkit.Wrap(err, "Failed to get root entry from kopia snapshot with ID", "backupId", backupID)
	}

	// Get the nested entry belonging to the backed up streaming file and return its object ID
	e, err := snapshotfs.GetNestedEntry(ctx, rootEntry, []string{filepath.Base(path)})
	if err != nil {
		return object.ID{}, errkit.Wrap(err, "Failed to get nested entry from kopia snapshot", "pathBase", filepath.Base(path))
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

// CustomConfigFileAndLogDirectory returns a config file path and log directory based on the hostname
func CustomConfigFileAndLogDirectory(hostname string) (string, string) {
	hostname = strings.ReplaceAll(hostname, ".", "-")
	configFile := filepath.Join(kopiacmd.DefaultConfigDirectory, hostname+".config")
	logDir := filepath.Join(kopiacmd.DefaultLogDirectory, hostname)
	return configFile, logDir
}
