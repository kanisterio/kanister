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

	// DataStoreParallelUploadName is the Environmental Variable set in Kanister
	// For Parallelism to be used by Kopia for backup action
	DataStoreParallelUploadName = "DATA_STORE_PARALLEL_UPLOAD"
	// DataStoreParallelFileOpsUploadsName is the Environmental Variable set in Kanister
	// for controlling the number of parallel file operations during snapshot
	DataStoreParallelFileOpsUploadsName = "DATA_STORE_PARALLEL_FILE_OPS_UPLOAD"
	// DataStoreParallelDirOpsUploadsName is the Environmental Variable set in Kanister
	// for controlling the number of parallel directory operations during snapshot
	DataStoreParallelDirOpsUploadsName = "DATA_STORE_PARALLEL_DIR_OPS_UPLOAD"
	// DefaultDataStoreParallelUpload is the Default Value of Parallelism
	DefaultDataStoreParallelUpload = 8
	// DefaultDataStoreParallelFileUploads is the Default Value of for file parallel operations.  Defaults to 0
	// which means share the goroutine pool controlled by DataStoreParallelUploadName
	DefaultDataStoreParallelFileUploads = 0
	// DefaultDataStoreParallelDirUploads is the Default Value of for directory parallel operations.  Defaults to 0
	// which means share the goroutine pool controlled by DataStoreParallelUploadName
	DefaultDataStoreParallelDirUploads = 0
	// DataStoreParallelDownloadName is the Environmental Variable set in Kanister
	// for Parallelism to be used by Kopia for restore action
	DataStoreParallelDownloadName = "DATA_STORE_PARALLEL_DOWNLOAD"
	// DefaultDataStoreParallelDownload is the Default Value of Parallelism
	DefaultDataStoreParallelDownload = 8
)

const (
	KopiaAPIServerAddressArg       = "serverAddress"
	KopiaTLSCertSecretKey          = "certs"
	KopiaTLSCertSecretDataArg      = "certData"
	KopiaServerPassphraseArg       = "serverPassphrase"
	KopiaServerPassphraseSecretKey = "serverPassphraseKey"
	KopiaUserPassphraseArg         = "userPassphrase"
)
