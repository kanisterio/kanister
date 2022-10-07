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
	gcSubCommand          = "gc"
	infoSubCommand        = "info"
	kopiaCommand          = "kopia"
	listSubCommand        = "list"
	maintenanceSubCommand = "maintenance"
	manifestSubCommand    = "manifest"
	policySubCommand      = "policy"
	restoreSubCommand     = "restore"
	runSubCommand         = "run"
	setSubCommand         = "set"
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
	logLevelErrorFlag          = "--log-level=error"
	logLevelInfoFlag           = "--log-level=info"
	noGrpcFlag                 = "--no-grpc"
	parallelFlag               = "--parallel"
	passwordFlag               = "--password"
	progressUpdateIntervalFlag = "--progress-update-interval"
	rawFlag                    = "--raw"
	showIdenticalFlag          = "--show-identical"
	tagsFlag                   = "--tags"
	unsafeIgnoreSourceFlag     = "--unsafe-ignore-source"
	ownerFlag                  = "--owner"
	sparseFlag                 = "--sparse"

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
	userPasswordFlag          = "--user-password"
)
