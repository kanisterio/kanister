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

package command

const (
	blobSubCommand        = "blob"
	createSubCommand      = "create"
	deleteSubCommand      = "delete"
	expireSubCommand      = "expire"
	infoSubCommand        = "info"
	kopiaCommand          = "kopia"
	listSubCommand        = "list"
	maintenanceSubCommand = "maintenance"
	manifestSubCommand    = "manifest"
	policySubCommand      = "policy"
	restoreSubCommand     = "restore"
	runSubCommand         = "run"
	setSubCommand         = "set"
	showSubCommand        = "show"
	snapshotSubCommand    = "snapshot"
	statsSubCommand       = "stats"

	allFlag                    = "--all"
	configFileFlag             = "--config-file"
	deleteFlag                 = "--delete"
	deltaFlag                  = "--delta"
	filterFlag                 = "--filter"
	globalFlag                 = "--global"
	jsonFlag                   = "--json"
	logDirectoryFlag           = "--log-dir"
	logLevelFlag               = "--log-level"
	fileLogLevelFlag           = "--file-log-level"
	LogLevelError              = "error"
	LogLevelInfo               = "info"
	parallelFlag               = "--parallel"
	passwordFlag               = "--password"
	progressUpdateIntervalFlag = "--progress-update-interval"
	rawFlag                    = "--raw"
	showIdenticalFlag          = "--show-identical"
	tagsFlag                   = "--tags"
	unsafeIgnoreSourceFlag     = "--unsafe-ignore-source"
	ownerFlag                  = "--owner"
	sparseFlag                 = "--write-sparse-files"
	ignorePermissionsError     = "--ignore-permission-errors"
	noIgnorePermissionsError   = "--no-ignore-permission-errors"

	// Server specific
	addSubCommand             = "add"
	refreshSubCommand         = "refresh"
	serverSubCommand          = "server"
	startSubCommand           = "start"
	statusSubCommand          = "status"
	setParametersSubCommand   = "set-parameters"
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
	userPasswordFlag          = "--user-password"
	enablePprof               = "--enable-pprof"
	metricsListerAddress      = "--metrics-listen-addr"
	htpasswdFilePath          = "--htpasswd-file"

	// Repository specific
	repositorySubCommand      = "repository"
	connectSubCommand         = "connect"
	noCheckForUpdatesFlag     = "--no-check-for-updates"
	overrideHostnameFlag      = "--override-hostname"
	overrideUsernameFlag      = "--override-username"
	pointInTimeConnectionFlag = "--point-in-time"
	urlFlag                   = "--url"
	readOnlyFlag              = "--readonly"
	retentionModeFlag         = "--retention-mode"
	retentionPeriodFlag       = "--retention-period"
)

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

	// Compression Algorithms recognized by Kopia
	s2DefaultComprAlgo = "s2-default"
)

// Constants for kopia defaults
const (
	// DefaultCacheDirectory is the directory where kopia content cache is created
	DefaultCacheDirectory = "/tmp/kopia-cache"

	// DefaultConfigFilePath is the file which contains kopia repo config
	DefaultConfigFilePath = "/tmp/kopia-repository.config"

	// DefaultConfigDirectory is the directory which contains custom kopia repo config
	DefaultConfigDirectory = "/tmp/kopia-repository"

	// DefaultLogDirectory is the directory where kopia log file is created
	DefaultLogDirectory = "/tmp/kopia-log"

	// DefaultHtpasswdFilePath is the path to the generated htpasswd file
	DefaultHtpasswdFilePath = "/tmp/kopia-htpasswd"
)
