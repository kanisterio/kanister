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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"


	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"github.com/jpillora/backoff"
	"github.com/kopia/kopia/repo"
	"github.com/kopia/kopia/repo/manifest"
	"github.com/kopia/kopia/repo/object"
	"github.com/kopia/kopia/snapshot"
	"github.com/kopia/kopia/snapshot/snapshotfs"

	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/format"
	kopiacmd "github.com/kanisterio/kanister/pkg/kopia/command"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/poll"
)

const (
	// DefaultDataStoreGeneralContentCacheSizeMB is the default content cache size for general command workloads
	DefaultDataStoreGeneralContentCacheSizeMB = 0

	// DefaultDataStoreGeneralMetadataCacheSizeMB is the default metadata cache size for general command workloads
	DefaultDataStoreGeneralMetadataCacheSizeMB = 500

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
	// ServerUsernameFormat is used to construct server username for Kopia API Server Status Command
	ServerUsernameFormat = "%s@%s"
	// KanisterAdminUsername is the username for the user with Admin privileges
	KanisterAdminUsername = "kanister-admin"
	defaultServerHostname = "data-mover-server-pod"
	// KanisterPodCustomLabelsEnv is the env var to get kanister pod custom labels
	KanisterPodCustomLabelsEnv = "KANISTER_POD_CUSTOM_LABELS"
	// KanisterPodCustomAnnotationsEnv is the env var to get kanister pod custom annotations
	KanisterPodCustomAnnotationsEnv = "KANISTER_POD_CUSTOM_ANNOTATIONS"

	// KanisterToolsMemoryRequestsEnv is the env var to get kanister sidecar or gvs restore data pod memory requests
	KanisterToolsMemoryRequestsEnv = "KANISTER_TOOLS_MEMORY_REQUESTS"
	// KanisterToolsCPURequestEnvs is the env var to get kanister sidecar or gvs restore data CPU requests
	KanisterToolsCPURequestsEnv = "KANISTER_TOOLS_CPU_REQUESTS"
	// KanisterToolsMemoryLimitsEnv is the env var to get kanister sidecar or gvs restore data memory limits
	KanisterToolsMemoryLimitsEnv = "KANISTER_TOOLS_MEMORY_LIMITS"
	// KanisterToolsCPULimitsEnv is the env var to get kanister sidecar or gvs restore data CPU limits
	KanisterToolsCPULimitsEnv = "KANISTER_TOOLS_CPU_LIMITS"
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
		return object.ID{}, errors.Wrapf(err, "Failed to load kopia snapshot with ID: %v", backupID)
	}

	// root entry of the kopia snapshot is a static directory with filepath.Dir(path) as its path
	if m.RootEntry == nil {
		return object.ID{}, errors.New("No root entry found in kopia manifest")
	}
	rootEntry, err := snapshotfs.SnapshotRoot(rep, m)
	if err != nil {
		return object.ID{}, errors.Wrapf(err, "Failed to get root entry from kopia snapshot with ID: %v", backupID)
	}

	// Get the nested entry belonging to the backed up streaming file and return its object ID
	e, err := snapshotfs.GetNestedEntry(ctx, rootEntry, []string{filepath.Base(path)})
	if err != nil {
		return object.ID{}, errors.Wrapf(err, "Failed to get nested entry from kopia snapshot: %v", filepath.Base(path))
	}

	return e.(object.HasObjectID).ObjectID(), nil
}

// GetDataStoreGeneralContentCacheSize finds and return content cache size from the options
func GetDataStoreGeneralContentCacheSize(opt map[string]int) int {
	if opt == nil {
		return DefaultDataStoreGeneralContentCacheSizeMB
	}
	if contentCacheSize, ok := opt[DataStoreGeneralContentCacheSizeMBKey]; ok {
		return contentCacheSize
	}
	return DefaultDataStoreGeneralContentCacheSizeMB
}

// GetDataStoreGeneralMetadataCacheSize finds and return metadata cache size from the options
func GetDataStoreGeneralMetadataCacheSize(opt map[string]int) int {
	if opt == nil {
		return DefaultDataStoreGeneralMetadataCacheSizeMB
	}
	if metadataCacheSize, ok := opt[DataStoreGeneralContentCacheSizeMBKey]; ok {
		return metadataCacheSize
	}
	return DefaultDataStoreGeneralMetadataCacheSizeMB
}

// GetCustomConfigFileAndLogDirectory returns a config file path and log directory based on the hostname
func GetCustomConfigFileAndLogDirectory(hostname string) (string, string) {
	hostname = strings.ReplaceAll(hostname, ".", "-")
	configFile := filepath.Join(kopiacmd.DefaultConfigDirectory, hostname+".config")
	logDir := filepath.Join(kopiacmd.DefaultLogDirectory, hostname)
	return configFile, logDir
}

// WaitTillCommandSucceed returns error if the Command fails to pass without error before default timeout
func WaitTillCommandSucceed(ctx context.Context, cli kubernetes.Interface, cmd []string, namespace, podName, container string) error {
	err := poll.WaitWithBackoff(ctx, backoff.Backoff{
		Factor: 2,
		Jitter: false,
		Min:    100 * time.Millisecond,
		Max:    180 * time.Second,
	}, func(context.Context) (bool, error) {
		stdout, stderr, exErr := kube.Exec(cli, namespace, podName, container, cmd, nil)
		format.Log(podName, container, stdout)
		format.Log(podName, container, stderr)
		if exErr != nil {
			return false, nil
		}
		return true, nil
	})
	return errors.Wrap(err, "Failed while waiting for Kopia API server to start")
}

// GetDefaultServerUsername returns the default server username used to run Kopia API Server commands
func GetDefaultServerUsername() string {
	return fmt.Sprintf(ServerUsernameFormat, KanisterAdminUsername, defaultServerHostname)
}

// SetLabelsToPodOptionsIfRequired sets labels to PodOptions
func SetLabelsToPodOptionsIfRequired(options *kube.PodOptions) {
	updateNeeded, labels := getKanisterPodLabels()
	if updateNeeded {
		if options.Labels == nil {
			options.Labels = make(map[string]string)
		}
		for k, v := range *labels {
			options.Labels[k] = v
		}
	}
}

func getKanisterPodLabels() (bool, *map[string]string) {
	return parseToLabelSelector(KanisterPodCustomLabelsEnv)
}

func parseToLabelSelector(envKey string) (bool, *map[string]string) {
	val, ok := os.LookupEnv(envKey)
	if !ok || val == "" {
		return false, nil
	}
	ls, err := metav1.ParseToLabelSelector(val)
	if err != nil {
		return false, nil
	}
	return true, &ls.MatchLabels
}

// SetAnnotationsToPodOptionsIfRequired sets annotations to PodOptions
func SetAnnotationsToPodOptionsIfRequired(options *kube.PodOptions) {
	updateNeeded, annotations := getKanisterPodAnnotations()
	if updateNeeded {
		if options.Annotations == nil {
			options.Annotations = make(map[string]string)
		}
		for k, v := range *annotations {
			options.Annotations[k] = v
		}
	}
}

func getKanisterPodAnnotations() (bool, *map[string]string) {
	return parseToLabelSelector(KanisterPodCustomAnnotationsEnv)
}

// SetResourceRequirementsToPodOptionsIfRequired sets resource requirements to PodOptions
func SetResourceRequirementsToPodOptionsIfRequired(options *kube.PodOptions) {
	updateNeeded, res := GetResourceRequirementsForKanisterPods()
	if updateNeeded {
		options.Resources = *res
	}
}

// GetResourceRequirementsForKanisterPods returns resource requirements if set in configmap
func GetResourceRequirementsForKanisterPods() (bool, *corev1.ResourceRequirements) {
	res := corev1.ResourceRequirements{
		Limits:   corev1.ResourceList{},
		Requests: corev1.ResourceList{},
	}
	updateNeeded := false
	resourceKeyValues := []string{
		KanisterToolsMemoryRequestsEnv,
		KanisterToolsCPURequestsEnv,
		KanisterToolsMemoryLimitsEnv,
		KanisterToolsCPULimitsEnv,
	}
	for _, key := range resourceKeyValues {
		val, ok := os.LookupEnv(key)
		if !ok || val == "" {
			continue
		}
		qty, err := resource.ParseQuantity(val)
		if err != nil {
			log.WithError(err)
			return false, nil
		}
		switch key {
		case KanisterToolsMemoryRequestsEnv:
			res.Requests[corev1.ResourceMemory] = qty
		case KanisterToolsCPURequestsEnv:
			res.Requests[corev1.ResourceCPU] = qty
		case KanisterToolsMemoryLimitsEnv:
			res.Limits[corev1.ResourceMemory] = qty
		case KanisterToolsCPULimitsEnv:
			res.Limits[corev1.ResourceCPU] = qty
		}
		updateNeeded = true
	}
	return updateNeeded, &res
}
