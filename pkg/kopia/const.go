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

const (
	// DefaultClientConfigFilePath is the file which contains kopia repo config
	DefaultClientConfigFilePath = "/tmp/kopia-repository.config"

	// DefaultClientCacheDirectory is the directory where kopia content cache is created
	DefaultClientCacheDirectory = "/tmp/kopia-cache"

	// DefaultDataStoreGeneralContentCacheSizeMB is the default content cache size for general command workloads
	DefaultDataStoreGeneralContentCacheSizeMB = 0
	// DataStoreGeneralContentCacheSizeMBVarName is the name of the environment variable that controls
	// kopia content cache size for general command workloads
	DataStoreGeneralContentCacheSizeMBVarName = "DATA_STORE_GENERAL_CONTENT_CACHE_SIZE_MB"

	// DefaultDataStoreGeneralMetadataCacheSizeMB is the default metadata cache size for general command workloads
	DefaultDataStoreGeneralMetadataCacheSizeMB = 500
	// DataStoreGeneralMetadataCacheSizeMBVarName is the name of the environment variable that controls
	// kopia metadata cache size for general command workloads
	DataStoreGeneralMetadataCacheSizeMBVarName = "DATA_STORE_GENERAL_METADATA_CACHE_SIZE_MB"

	// DefaultDataStoreRestoreContentCacheSizeMB is the default content cache size for restore workloads
	DefaultDataStoreRestoreContentCacheSizeMB = 500
	// DataStoreRestoreContentCacheSizeMBVarName is the name of the environment variable that controls
	// kopia content cache size for restore workloads
	DataStoreRestoreContentCacheSizeMBVarName = "DATA_STORE_RESTORE_CONTENT_CACHE_SIZE_MB"

	// DefaultDataStoreRestoreMetadataCacheSizeMB is the default metadata cache size for restore workloads
	DefaultDataStoreRestoreMetadataCacheSizeMB = 500
	// DataStoreRestoreMetadataCacheSizeMBVarName is the name of the environment variable that controls
	// kopia metadata cache size for restore workloads
	DataStoreRestoreMetadataCacheSizeMBVarName = "DATA_STORE_RESTORE_METADATA_CACHE_SIZE_MB"

	// DefaultDataStoreParallelUpload is the default value for data store parallelism
	DefaultDataStoreParallelUpload = 8

	// DataStoreParallelUploadVarName is the name of the environment variable that controls
	// kopia parallelism during snapshot create commands
	DataStoreParallelUploadVarName = "DATA_STORE_PARALLEL_UPLOAD"

	ManifestTypeSnapshotFilter = "type:snapshot"
)

const (
	KopiaAPIServerAddressArg       = "serverAddress"
	KopiaTLSCertSecretKey          = "certs"
	KopiaTLSCertSecretDataArg      = "certData"
	KopiaServerPassphraseArg       = "serverPassphrase"
	KopiaServerPassphraseSecretKey = "serverPassphraseKey"
	KopiaUserPassphraseArg         = "userPassphrase"
)
