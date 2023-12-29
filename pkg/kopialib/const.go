// Copyright 2023 The Kanister Authors.
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
	// S3 storage consts
	BucketKey         = "bucket"
	EndpointKey       = "endpoint"
	PrefixKey         = "prefix"
	RegionKey         = "region"
	SkipSSLVerifyKey  = "skipSSLVerify"
	S3AccessKey       = "accessKeyID"
	S3SecretAccessKey = "secretAccessKey"
	S3TokenKey        = "sessionToken"
	DoNotUseTLS       = "doNotUseTLS"
	DoNotVerifyTLS    = "doNotVerifyTLS"

	//Azure storage conts
	AzureStorageAccount          = "storageAccount"
	AzureStorageAccountAccessKey = "storageKey"
	AzureSASToken                = "sasToken"
)
