// Copyright 2024 The Kanister Authors.
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

package kopialib

const (
	// Common storage consts
	BucketKey = "bucket"
	PrefixKey = "prefix"

	// S3 storage consts
	S3EndpointKey     = "endpoint"
	S3RegionKey       = "region"
	SkipSSLVerifyKey  = "skipSSLVerify"
	S3AccessKey       = "accessKeyID"
	S3SecretAccessKey = "secretAccessKey"
	S3TokenKey        = "sessionToken"
	DoNotUseTLS       = "doNotUseTLS"
	DoNotVerifyTLS    = "doNotVerifyTLS"

	// Azure storage consts
	AzureStorageAccount          = "storageAccount"
	AzureStorageAccountAccessKey = "storageKey"
	AzureSASToken                = "sasToken"

	// Filestore storage consts
	FilesystorePath    = "path"
	DefaultFSMountPath = "/mnt/data"

	// GCP storage consts
	GCPServiceAccountCredentialsFile = "serviceAccountCredentialsFile"
	GCPServiceAccountCredentialJSON  = "serviceAccountCredentialsJson"
	GCPReadOnly                      = "readOnly"
)
