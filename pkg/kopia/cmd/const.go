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

package cmd

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

	snapshotTypeField = "type:snapshot"

	// List of possible modifications to a policy, expressed as the kopia flag that will modify it
	// Retention
	keepLatest  = "--keep-latest"
	keepHourly  = "--keep-hourly"
	keepDaily   = "--keep-daily"
	keepWeekly  = "--keep-weekly"
	keepMonthly = "--keep-monthly"
	keepAnnual  = "--keep-annual"
	// Compression
	compressionAlgorithm = "--compression"

	// List of kopia-supported compression algorithms recognized by the kopia "--compression" flag
	s2DefaultComprAlgo = "s2-default"
)
