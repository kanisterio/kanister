// Copyright 2022 The Kanister Authors.
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
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/pkg/errors"

	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/logsafe"
	"github.com/kanisterio/kanister/pkg/utils"
)

const (
	blobSubCommand        = "blob"
	connectSubCommand     = "connect"
	createSubCommand      = "create"
	deleteSubCommand      = "delete"
	expireSubCommand      = "expire"
	gcSubCommand          = "gc"
	infoSubCommand        = "info"
	kopiaCommand          = "kopia"
	listSubCommand        = "list"
	maintenanceSubCommand = "maintenance"
	manifestSubCommand    = "manifest"
	policySubCommand      = "policy"
	repositorySubCommand  = "repository"
	restoreSubCommand     = "restore"
	runSubCommand         = "run"
	setSubCommand         = "set"
	snapshotSubCommand    = "snapshot"
	statsSubCommand       = "stats"

	allFlag                    = "--all"
	bucketFlag                 = "--bucket"
	cacheDirectoryFlag         = "--cache-directory"
	configFileFlag             = "--config-file"
	contentCacheSizeMBFlag     = "--content-cache-size-mb"
	deleteFlag                 = "--delete"
	deltaFlag                  = "--delta"
	endpointFlag               = "--endpoint"
	filterFlag                 = "--filter"
	globalFlag                 = "--global"
	jsonFlag                   = "--json"
	logDirectoryFlag           = "--log-dir"
	logLevelErrorFlag          = "--log-level=error"
	logLevelInfoFlag           = "--log-level=info"
	metadataCacheSizeMBFlag    = "--metadata-cache-size-mb"
	noCheckForUpdatesFlag      = "--no-check-for-updates"
	noGrpcFlag                 = "--no-grpc"
	noProgressFlag             = "--no-progress"
	overrideHostnameFlag       = "--override-hostname"
	overrideUsernameFlag       = "--override-username"
	parallelFlag               = "--parallel"
	passwordFlag               = "--password"
	pointInTimeConnectionFlag  = "--point-in-time"
	prefixFlag                 = "--prefix"
	progressUpdateIntervalFlag = "--progress-update-interval"
	rawFlag                    = "--raw"
	showIdenticalFlag          = "--show-identical"
	unsafeIgnoreSourceFlag     = "--unsafe-ignore-source"
	ownerFlag                  = "--owner"
	sparseFlag                 = "--sparse"

	// S3 specific
	s3SubCommand         = "s3"
	accessKeyFlag        = "--access-key"
	disableTLSFlag       = "--disable-tls"
	disableTLSVerifyFlag = "--disable-tls-verification"
	secretAccessKeyFlag  = "--secret-access-key"
	sessionTokenFlag     = "--session-token"
	regionFlag           = "--region"

	// Azure specific
	azureSubCommand    = "azure"
	containerFlag      = "--container"
	storageAccountFlag = "--storage-account"
	storageKeyFlag     = "--storage-key"
	storageDomainFlag  = "--storage-domain"

	// Google specific
	googleSubCommand    = "gcs"
	credentialsFileFlag = "--credentials-file"

	// Server specific
	addSubCommand             = "add"
	refreshSubCommand         = "refresh"
	serverSubCommand          = "server"
	startSubCommand           = "start"
	statusSubCommand          = "status"
	userSubCommand            = "user"
	addressFlag               = "--address"
	redirectToDevNull         = "> /dev/null 2>&1"
	runInBackground           = "&"
	serverControlPasswordFlag = "--server-control-password"
	serverControlUsernameFlag = "--server-control-username"
	serverPasswordFlag        = "--server-password"
	serverUsernameFlag        = "--server-username"
	serverCertFingerprint     = "--server-cert-fingerprint"
	tlsCertFilePath           = "--tls-cert-file"
	tlsGenerateCertFlag       = "--tls-generate-cert"
	tlsKeyFilePath            = "--tls-key-file"
	urlFlag                   = "--url"
	userPasswordFlag          = "--user-password"

	// Filesystem specific
	filesystemSubCommand = "filesystem"
	pathFlag             = "--path"

	// DefaultCacheDirectory is the directory where kopia content cache is created
	DefaultCacheDirectory = "/tmp/kopia-cache"

	// DefaultConfigFilePath is the file which contains kopia repo config
	DefaultConfigFilePath = "/tmp/kopia-repository.config"

	// DefaultConfigDirectory is the directory which contains custom kopia repo config
	DefaultConfigDirectory = "/tmp/kopia-repository"

	// DefaultLogDirectory is the directory where kopia log file is created
	DefaultLogDirectory = "/tmp/kopia-log"

	// DefaultSparseRestore is the default option for whether to do a sparse restore
	DefaultSparseRestore = false

	// DefaultFSMountPath is the mount path for the file store PVC on Kopia API server
	DefaultFSMountPath = "/mnt/data"

	// Filters
	manifestTypeSnapshotFilter = "type:snapshot"

	// DefaultK10DataStoreGeneralContentCacheSizeMB is the default content cache size for general command workloads
	DefaultK10DataStoreGeneralContentCacheSizeMB = 0
	// K10DataStoreGeneralContentCacheSizeMBVarName is the name of the environment variable that controls
	// kopia content cache size for general command workloads
	K10DataStoreGeneralContentCacheSizeMBVarName = "K10_DATA_STORE_GENERAL_CONTENT_CACHE_SIZE_MB"

	// DefaultK10DataStoreGeneralMetadataCacheSizeMB is the default metadata cache size for general command workloads
	DefaultK10DataStoreGeneralMetadataCacheSizeMB = 500
	// K10DataStoreGeneralMetadataCacheSizeMBVarName is the name of the environment variable that controls
	// kopia metadata cache size for general command workloads
	K10DataStoreGeneralMetadataCacheSizeMBVarName = "K10_DATA_STORE_GENERAL_METADATA_CACHE_SIZE_MB"

	// DefaultK10DataStoreRestoreContentCacheSizeMB is the default content cache size for restore workloads
	DefaultK10DataStoreRestoreContentCacheSizeMB = 500
	// K10DataStoreRestoreContentCacheSizeMBVarName is the name of the environment variable that controls
	// kopia content cache size for restore workloads
	K10DataStoreRestoreContentCacheSizeMBVarName = "K10_DATA_STORE_RESTORE_CONTENT_CACHE_SIZE_MB"

	// DefaultK10DataStoreRestoreMetadataCacheSizeMB is the default metadata cache size for restore workloads
	DefaultK10DataStoreRestoreMetadataCacheSizeMB = 500
	// K10DataStoreRestoreMetadataCacheSizeMBVarName is the name of the environment variable that controls
	// kopia metadata cache size for restore workloads
	K10DataStoreRestoreMetadataCacheSizeMBVarName = "K10_DATA_STORE_RESTORE_METADATA_CACHE_SIZE_MB"

	// DefaultK10DataStoreParallelUpload is the default value for data store parallelism
	DefaultK10DataStoreParallelUpload = 8
	// K10DataStoreParallelUploadVarName is the name of the environment variable that controls
	// kopia parallelism during snapshot create commands
	K10DataStoreParallelUploadVarName = "K10_DATA_STORE_PARALLEL_UPLOAD"
)

func bashCommand(args logsafe.Cmd) []string {
	log.Info().Print("Kopia Command", field.M{"Command": args.String()})
	return []string{"bash", "-o", "errexit", "-c", args.PlainText()}
}

func stringSliceCommand(args logsafe.Cmd) []string {
	log.Info().Print("Kopia Command", field.M{"Command": args.String()})
	return args.StringSliceCMD()
}

func kopiaArgs(password, configFilePath, logDirectory string, requireInfoLevel bool) logsafe.Cmd {
	c := logsafe.NewLoggable(kopiaCommand)
	if requireInfoLevel {
		c = c.AppendLoggable(logLevelInfoFlag)
	} else {
		c = c.AppendLoggable(logLevelErrorFlag)
	}
	if configFilePath != "" {
		c = c.AppendLoggableKV(configFileFlag, configFilePath)
	}
	if logDirectory != "" {
		c = c.AppendLoggableKV(logDirectoryFlag, logDirectory)
	}
	if password != "" {
		c = c.AppendRedactedKV(passwordFlag, password)
	}
	return c
}

// ExecKopiaArgs returns the basic Argv for executing kopia with the given config file path.
func ExecKopiaArgs(configFilePath string) []string {
	return kopiaArgs("", configFilePath, "", false).StringSliceCMD()
}

func kopiaCacheArgs(args logsafe.Cmd, cacheDirectory string, contentCacheMB, metadataCacheMB int) logsafe.Cmd {
	args = args.AppendLoggableKV(cacheDirectoryFlag, cacheDirectory)
	args = args.AppendLoggableKV(contentCacheSizeMBFlag, strconv.Itoa(contentCacheMB))
	args = args.AppendLoggableKV(metadataCacheSizeMBFlag, strconv.Itoa(metadataCacheMB))
	return args
}

// GetCacheSizeSettingsForSnapshot returns the feature setting cache size values to be used
// for initializing repositories that will be performing general command workloads that benefit from
// cacheing metadata only.
func GetCacheSizeSettingsForSnapshot() (contentCacheMB, metadataCacheMB int) {
	return utils.GetEnvAsIntOrDefault(K10DataStoreGeneralContentCacheSizeMBVarName, DefaultK10DataStoreGeneralContentCacheSizeMB),
		utils.GetEnvAsIntOrDefault(K10DataStoreGeneralMetadataCacheSizeMBVarName, DefaultK10DataStoreGeneralMetadataCacheSizeMB)
}

// GetCacheSizeSettingsForRestore returns the feature setting cache size values to be used
// for initializing repositories that will be performing restore workloads
func GetCacheSizeSettingsForRestore() (contentCacheMB, metadataCacheMB int) {
	return utils.GetEnvAsIntOrDefault(K10DataStoreRestoreContentCacheSizeMBVarName, DefaultK10DataStoreRestoreContentCacheSizeMB),
		utils.GetEnvAsIntOrDefault(K10DataStoreRestoreMetadataCacheSizeMBVarName, DefaultK10DataStoreRestoreMetadataCacheSizeMB)
}

// SnapshotCreateCommand returns the kopia command for creation of a snapshot
// TODO: Have better mechanism to apply global flags
func SnapshotCreateCommand(encryptionKey, pathToBackup, configFilePath, logDirectory string) []string {
	return stringSliceCommand(snapshotCreateCommand(encryptionKey, pathToBackup, configFilePath, logDirectory))
}

func snapshotCreateCommand(encryptionKey, pathToBackup, configFilePath, logDirectory string) logsafe.Cmd {
	const (
		// kube.Exec might timeout after 4h if there is no output from the command
		// Setting it to 1h instead of 1000000h so that kopia logs progress once every hour
		longUpdateInterval = "1h"

		requireLogLevelInfo = true
	)

	parallelismStr := strconv.Itoa(utils.GetEnvAsIntOrDefault(K10DataStoreParallelUploadVarName, DefaultK10DataStoreParallelUpload))
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, requireLogLevelInfo)
	args = args.AppendLoggable(snapshotSubCommand, createSubCommand, pathToBackup, jsonFlag)
	args = args.AppendLoggableKV(parallelFlag, parallelismStr)
	args = args.AppendLoggableKV(progressUpdateIntervalFlag, longUpdateInterval)

	return args
}

// SnapshotCreateInfo is a container for data that can be parsed from the output of
// `kopia snapshot create`.
type SnapshotCreateInfo struct {
	SnapshotID string
	RootID     string
	Stats      *SnapshotCreateStats
}

// ParseSnapshotCreateOutput parses the output of a snapshot create command into
// a new SnapshotCreateInfo struct and returns its pointer. The Stats field may be nil
// if the stats were unable to be parsed. The root ID and snapshot ID are fetched from
// structured stdout and stats are parsed from stderr output.
func ParseSnapshotCreateOutput(snapCreateStdoutOutput, snapCreateStderrOutput string) (*SnapshotCreateInfo, error) {
	snapID, rootID, err := SnapshotInfoFromSnapshotCreateOutput(snapCreateStdoutOutput)
	if err != nil {
		return nil, err
	}

	return &SnapshotCreateInfo{
		SnapshotID: snapID,
		RootID:     rootID,
		Stats:      SnapshotStatsFromSnapshotCreate(snapCreateStderrOutput),
	}, nil
}

// SnapshotCreateStats is a container for stats parsed from the output of a `kopia
// snapshot create` command.
type SnapshotCreateStats struct {
	FilesHashed   int64
	SizeHashedB   int64
	FilesCached   int64
	SizeCachedB   int64
	SizeUploadedB int64
}

// SnapshotStatsFromSnapshotCreate parses the output of a kopia snapshot
// create execution for a log of the stats for that execution.
func SnapshotStatsFromSnapshotCreate(snapCreateStderrOutput string) (stats *SnapshotCreateStats) {
	if snapCreateStderrOutput == "" {
		return nil
	}
	logs := regexp.MustCompile("[\r\n]").Split(snapCreateStderrOutput, -1)

	// Match a pattern starting with "*" (signifying upload finished), and containing
	// the repeated pattern "<\d+> <type> (<humanized size base 10>),",
	// where <type> is "hashed", "cached", and "uploaded".
	// Example input:
	// 	 * 0 hashing, 1 hashed (2 B), 3 cached (40 KB), uploaded 6.7 GB, estimated 1092.3 MB (100.0%) 0s left
	// Expected output:
	// SnapshotCreateStats{
	// 		filesHashed:  1,
	// 		sizeHashedB: 2,
	// 		filesCached:  3,
	// 		sizeCachedB: 40000,
	// 		sizeUploadedB: 6700000000,
	// }, nil
	pattern := regexp.MustCompile(`\*.+[^\d](\d+) hashed \(([^\)]+)\), (\d+) cached \(([^\)]+)\), uploaded ([^\)]+),.+`)
	for _, l := range logs {
		match := pattern.FindStringSubmatch(l)
		if match != nil && len(match) >= 6 {
			numHashedStr := match[1]
			hashedSizeHumanized := match[2]
			numCachedStr := match[3]
			cachedSizeHumanized := match[4]
			uploadedSizeHumanized := match[5]

			numHashed, err := strconv.Atoi(numHashedStr)
			if err != nil {
				log.WithError(err).Print("Skipping entry due to inability to parse number of hashed files", field.M{"numHashedStr": numHashedStr})
				continue
			}

			numCached, err := strconv.Atoi(numCachedStr)
			if err != nil {
				log.WithError(err).Print("Skipping entry due to inability to parse number of cached files", field.M{"numCachedStr": numCachedStr})
				continue
			}

			hashedSizeBytes, err := humanize.ParseBytes(hashedSizeHumanized)
			if err != nil {
				log.WithError(err).Print("Skipping entry due to inability to parse hashed size string", field.M{"hashedSizeHumanized": hashedSizeHumanized})
				continue
			}

			cachedSizeBytes, err := humanize.ParseBytes(cachedSizeHumanized)
			if err != nil {
				log.WithError(err).Print("Skipping entry due to inability to parse cached size string", field.M{"cachedSizeHumanized": cachedSizeHumanized})
				continue
			}

			uploadedSizeBytes, err := humanize.ParseBytes(uploadedSizeHumanized)
			if err != nil {
				log.WithError(err).Print("Skipping entry due to inability to parse uploaded size string", field.M{"uploadedSizeHumanized": uploadedSizeHumanized})
				continue
			}

			stats = &SnapshotCreateStats{
				FilesHashed:   int64(numHashed),
				SizeHashedB:   int64(hashedSizeBytes),
				FilesCached:   int64(numCached),
				SizeCachedB:   int64(cachedSizeBytes),
				SizeUploadedB: int64(uploadedSizeBytes),
			}
		}
	}

	if stats == nil {
		log.Error().Print("could not find well-formed stats in snapshot create output")
	}

	return stats
}

// RepoSizeStatsFromBlobStatsRaw takes a string as input, interprets it as a kopia blob stats
// output in an expected format (Contains the line "Total: <size>"), and returns the integer
// size in bytes or an error if parsing is unsuccessful.
func RepoSizeStatsFromBlobStatsRaw(blobStats string) (phySizeTotal int64, blobCount int, err error) {
	if blobStats == "" {
		return phySizeTotal, blobCount, errors.New("received empty blob stats string")
	}

	sizePattern := regexp.MustCompile(`Total: (\d+)$`)
	countPattern := regexp.MustCompile(`Count: (\d+)$`)

	var countStr, sizeStr string

	for _, l := range strings.Split(blobStats, "\n") {
		if countStr == "" {
			countMatch := countPattern.FindStringSubmatch(l)
			if countMatch != nil && len(countMatch) >= 2 {
				countStr = countMatch[1]
			}
		}

		if sizeStr == "" {
			sizeMatch := sizePattern.FindStringSubmatch(l)
			if sizeMatch != nil && len(sizeMatch) >= 2 {
				sizeStr = sizeMatch[1]
			}
		}

		if !(countStr == "" || sizeStr == "") {
			// Both strings have been matched
			break
		}
	}

	if countStr == "" {
		return phySizeTotal, blobCount, errors.New("could not find count field in the blob stats")
	}

	if sizeStr == "" {
		return phySizeTotal, blobCount, errors.New("could not find size field in the blob stats")
	}

	countVal, err := strconv.Atoi(countStr)
	if err != nil {
		return phySizeTotal, blobCount, errors.Wrap(err, fmt.Sprintf("unable to convert parsed count value %s", countStr))
	}

	sizeValBytes, err := strconv.Atoi(sizeStr)
	if err != nil {
		return phySizeTotal, blobCount, errors.Wrap(err, fmt.Sprintf("unable to convert parsed size value %s", countStr))
	}

	return int64(sizeValBytes), countVal, nil
}

// SnapshotExpireCommand returns the kopia command for removing snapshots with given root ID
func SnapshotExpireCommand(encryptionKey, rootID, configFilePath, logDirectory string, mustDelete bool) []string {
	return stringSliceCommand(snapshotExpireCommand(encryptionKey, rootID, configFilePath, logDirectory, mustDelete))
}

func snapshotExpireCommand(encryptionKey, rootID, configFilePath, logDirectory string, mustDelete bool) logsafe.Cmd {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(snapshotSubCommand, expireSubCommand, rootID)
	if mustDelete {
		args = args.AppendLoggable(deleteFlag)
	}

	return args
}

// SnapshotRestoreCommand returns kopia command restoring snapshots with given snap ID
func SnapshotRestoreCommand(encryptionKey, snapID, targetPath, configFilePath, logDirectory string, sparseRestore bool) []string {
	return stringSliceCommand(snapshotRestoreCommand(encryptionKey, snapID, targetPath, configFilePath, logDirectory, sparseRestore))
}

func snapshotRestoreCommand(encryptionKey, snapID, targetPath, configFilePath, logDirectory string, sparseRestore bool) logsafe.Cmd {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(snapshotSubCommand, restoreSubCommand, snapID, targetPath)
	if sparseRestore {
		args = args.AppendLoggable(sparseFlag)
	}

	return args
}

// RestoreCommand returns the kopia command for restoring root of a snapshot with given root ID
func RestoreCommand(encryptionKey, rootID, targetPath, configFilePath, logDirectory string) []string {
	return stringSliceCommand(restoreCommand(encryptionKey, rootID, targetPath, configFilePath, logDirectory))
}

func restoreCommand(encryptionKey, rootID, targetPath, configFilePath, logDirectory string) logsafe.Cmd {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(restoreSubCommand, rootID, targetPath)

	return args
}

// DeleteCommand returns the kopia command for deleting a snapshot with given snapshot ID
func DeleteCommand(encryptionKey, snapID, configFilePath, logDirectory string) []string {
	return stringSliceCommand(deleteCommand(encryptionKey, snapID, configFilePath, logDirectory))
}

func deleteCommand(encryptionKey, snapID, configFilePath, logDirectory string) logsafe.Cmd {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(snapshotSubCommand, deleteSubCommand, snapID, unsafeIgnoreSourceFlag)

	return args
}

// SnapshotGCCommand returns the kopia command for issuing kopia snapshot gc
func SnapshotGCCommand(encryptionKey, configFilePath, logDirectory string) []string {
	return stringSliceCommand(snapshotGCCommand(encryptionKey, configFilePath, logDirectory))
}

func snapshotGCCommand(encryptionKey, configFilePath, logDirectory string) logsafe.Cmd {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(snapshotSubCommand, gcSubCommand, deleteFlag)

	return args
}

// MaintenanceSetCommandWithOwner returns the kopia command for setting custom maintenance owner
func MaintenanceSetCommandWithOwner(encryptionKey, configFilePath, logDirectory, customOwner string) []string {
	return stringSliceCommand(maintenanceSetOwner(encryptionKey, configFilePath, logDirectory, customOwner))
}

func maintenanceSetOwner(encryptionKey, configFilePath, logDirectory, customOwner string) logsafe.Cmd {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(maintenanceSubCommand, setSubCommand)
	args = args.AppendLoggableKV(ownerFlag, customOwner)
	return args
}

// MaintenanceRunCommand returns the kopia command to run manual maintenance
func MaintenanceRunCommand(encryptionKey, configFilePath, logDirectory string) []string {
	return stringSliceCommand(maintenanceRunCommand(encryptionKey, configFilePath, logDirectory))
}

func maintenanceRunCommand(encryptionKey, configFilePath, logDirectory string) logsafe.Cmd {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(maintenanceSubCommand, runSubCommand)

	return args
}

// MaintenanceInfoCommand returns the kopia command to get maintenance info
func MaintenanceInfoCommand(encryptionKey, configFilePath, logDirectory string) []string {
	return stringSliceCommand(maintenanceInfoCommand(encryptionKey, configFilePath, logDirectory))
}

func maintenanceInfoCommand(encryptionKey, configFilePath, logDirectory string) logsafe.Cmd {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(maintenanceSubCommand, infoSubCommand)

	return args
}

// BlobList returns the kopia command for listing blobs in the repository with their sizes
func BlobList(encryptionKey, configFilePath, logDirectory string) []string {
	return stringSliceCommand(blobList(encryptionKey, configFilePath, logDirectory))
}

func blobList(encryptionKey, configFilePath, logDirectory string) logsafe.Cmd {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(blobSubCommand, listSubCommand)

	return args
}

// BlobStats returns the kopia command to get the blob stats
func BlobStats(encryptionKey, configFilePath, logDirectory string) []string {
	return stringSliceCommand(blobStats(encryptionKey, configFilePath, logDirectory))
}

func blobStats(encryptionKey, configFilePath, logDirectory string) logsafe.Cmd {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(blobSubCommand, statsSubCommand, rawFlag)

	return args
}

// SnapListAll returns the kopia command for listing all snapshots in the repository with their sizes
func SnapListAll(encryptionKey, configFilePath, logDirectory string) []string {
	return stringSliceCommand(snapListAll(encryptionKey, configFilePath, logDirectory))
}

func snapListAll(encryptionKey, configFilePath, logDirectory string) logsafe.Cmd {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(
		snapshotSubCommand,
		listSubCommand,
		allFlag,
		deltaFlag,
		showIdenticalFlag,
		jsonFlag,
	)

	return args
}

// SnapListAllWithSnapIDs returns the kopia command for listing all snapshots in the repository with snapshotIDs
func SnapListAllWithSnapIDs(encryptionKey, configFilePath, logDirectory string) []string {
	return stringSliceCommand(snapListAllWithSnapIDs(encryptionKey, configFilePath, logDirectory))
}

func snapListAllWithSnapIDs(encryptionKey, configFilePath, logDirectory string) logsafe.Cmd {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(manifestSubCommand, listSubCommand, jsonFlag)
	args = args.AppendLoggableKV(filterFlag, manifestTypeSnapshotFilter)

	return args
}

// ServerCommand returns the kopia command for starting the Kopia API Server
func ServerStartCommand(
	configFilePath,
	logDirectory,
	serverAddress,
	tlsCertFile,
	tlsKeyFile,
	serverUsername,
	serverPassword string,
	autoGenerateCert,
	background bool,
) []string {
	return bashCommand(serverStartCommand(
		configFilePath,
		logDirectory,
		serverAddress,
		tlsCertFile,
		tlsKeyFile,
		serverUsername,
		serverPassword,
		autoGenerateCert,
		background,
	))
}

func serverStartCommand(
	configFilePath,
	logDirectory,
	serverAddress,
	tlsCertFile,
	tlsKeyFile,
	serverUsername,
	serverPassword string,
	autoGenerateCert,
	background bool,
) logsafe.Cmd {
	args := kopiaArgs("", configFilePath, logDirectory, false)

	if autoGenerateCert {
		args = args.AppendLoggable(serverSubCommand, startSubCommand, tlsGenerateCertFlag)
	} else {
		args = args.AppendLoggable(serverSubCommand, startSubCommand)
	}
	args = args.AppendLoggableKV(addressFlag, serverAddress)
	args = args.AppendLoggableKV(tlsCertFilePath, tlsCertFile)
	args = args.AppendLoggableKV(tlsKeyFilePath, tlsKeyFile)
	args = args.AppendLoggableKV(serverUsernameFlag, serverUsername)
	args = args.AppendRedactedKV(serverPasswordFlag, serverPassword)

	args = args.AppendLoggableKV(serverControlUsernameFlag, serverUsername)
	args = args.AppendRedactedKV(serverControlPasswordFlag, serverPassword)

	// TODO: Remove when GRPC support is added
	args = args.AppendLoggable(noGrpcFlag)

	if background {
		// To start the server and run in the background
		args = args.AppendLoggable(redirectToDevNull, runInBackground)
	}

	return args
}

// ServerStatusCommand returns the kopia command for checking status of the Kopia API Server
func ServerStatusCommand(
	configFilePath,
	logDirectory,
	serverAddress,
	serverUsername,
	serverPassword,
	fingerprint string,
) []string {
	return stringSliceCommand(serverStatusCommand(
		configFilePath,
		logDirectory,
		serverAddress,
		serverUsername,
		serverPassword,
		fingerprint,
	))
}

func serverStatusCommand(
	configFilePath,
	logDirectory,
	serverAddress,
	serverUsername,
	serverPassword,
	fingerprint string,
) logsafe.Cmd {
	args := kopiaArgs("", configFilePath, logDirectory, false)
	args = args.AppendLoggable(serverSubCommand, statusSubCommand)
	args = args.AppendLoggableKV(addressFlag, serverAddress)
	args = args.AppendRedactedKV(serverCertFingerprint, fingerprint)
	args = args.AppendLoggableKV(serverUsernameFlag, serverUsername)
	args = args.AppendRedactedKV(serverPasswordFlag, serverPassword)

	return args
}

// ServerAddUserCommand returns the kopia command adding a new user to the Kopia API Server
func ServerAddUserCommand(
	encryptionKey,
	configFilePath,
	logDirectory,
	newUsername,
	userPassword string,
) []string {
	return stringSliceCommand(serverAddUserCommand(
		encryptionKey,
		configFilePath,
		logDirectory,
		newUsername,
		userPassword,
	))
}

func serverAddUserCommand(
	encryptionKey,
	configFilePath,
	logDirectory,
	newUsername,
	userPassword string,
) logsafe.Cmd {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(serverSubCommand, userSubCommand, addSubCommand, newUsername)
	args = args.AppendRedactedKV(userPasswordFlag, userPassword)

	return args
}

// ServerSetUserCommand returns the kopia command setting password for existing user for the Kopia API Server
func ServerSetUserCommand(
	encryptionKey,
	configFilePath,
	logDirectory,
	newUsername,
	userPassword string,
) []string {
	return stringSliceCommand(serverSetUserCommand(
		encryptionKey,
		configFilePath,
		logDirectory,
		newUsername,
		userPassword,
	))
}

func serverSetUserCommand(
	encryptionKey,
	configFilePath,
	logDirectory,
	newUsername,
	userPassword string,
) logsafe.Cmd {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(serverSubCommand, userSubCommand, setSubCommand, newUsername)
	args = args.AppendRedactedKV(userPasswordFlag, userPassword)

	return args
}

// ServerListUserCommand returns the kopia command to list users from the Kopia API Server
func ServerListUserCommand(
	encryptionKey,
	configFilePath,
	logDirectory string,
) []string {
	return stringSliceCommand(serverListUserCommand(
		encryptionKey,
		configFilePath,
		logDirectory,
	))
}

func serverListUserCommand(
	encryptionKey,
	configFilePath,
	logDirectory string,
) logsafe.Cmd {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(serverSubCommand, userSubCommand, listSubCommand, jsonFlag)

	return args
}

// ServerRefreshCommand returns the kopia command for refreshing the Kopia API Server
// This helps allow new users to be able to connect to the Server instead of waiting for auto-refresh
func ServerRefreshCommand(
	encryptionKey,
	configFilePath,
	logDirectory,
	serverAddress,
	serverUsername,
	serverPassword,
	fingerprint string,
) []string {
	return stringSliceCommand(serverRefreshCommand(
		encryptionKey,
		configFilePath,
		logDirectory,
		serverAddress,
		serverUsername,
		serverPassword,
		fingerprint,
	))
}

func serverRefreshCommand(
	encryptionKey,
	configFilePath,
	logDirectory,
	serverAddress,
	serverUsername,
	serverPassword,
	fingerprint string,
) logsafe.Cmd {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(serverSubCommand, refreshSubCommand)
	args = args.AppendRedactedKV(serverCertFingerprint, fingerprint)
	args = args.AppendLoggableKV(addressFlag, serverAddress)
	args = args.AppendLoggableKV(serverUsernameFlag, serverUsername)
	args = args.AppendRedactedKV(serverPasswordFlag, serverPassword)

	return args
}

const (
	snapshotTypeField = "type:snapshot"
)

type policyChanges map[string]string

// List of possible modifications to a policy, expressed as the kopia flag that will modify it
const (
	// Retention
	keepLatest  = "--keep-latest"
	keepHourly  = "--keep-hourly"
	keepDaily   = "--keep-daily"
	keepWeekly  = "--keep-weekly"
	keepMonthly = "--keep-monthly"
	keepAnnual  = "--keep-annual"

	// Compression
	compressionAlgorithm = "--compression"
)

// List of kopia-supported compression algorithms recognized by the kopia "--compression" flag
const (
	s2DefaultComprAlgo = "s2-default"
)

// policySetGlobalCommand returns the kopia command for modifying the global policy
func policySetGlobalCommand(encryptionKey, configFilePath, logDirectory string, modifications policyChanges) []string {
	return stringSliceCommand(policySetGlobalCommandSetup(encryptionKey, configFilePath, logDirectory, modifications))
}

func policySetGlobalCommandSetup(encryptionKey, configFilePath, logDirectory string, modifications policyChanges) logsafe.Cmd {
	args := kopiaArgs(encryptionKey, configFilePath, logDirectory, false)
	args = args.AppendLoggable(policySubCommand, setSubCommand, globalFlag)
	for field, val := range modifications {
		args = args.AppendLoggableKV(field, val)
	}

	return args
}
