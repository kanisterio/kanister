// Copyright 2019 The Kanister Authors.
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

package objectstore

const googleGCSHost = "https://storage.googleapis.com"
const REGIONAL = "REGIONAL"

// ProviderType enum for different providers
type ProviderType string

const (
	// ProviderTypeGCS captures enum value "GCS"
	ProviderTypeGCS ProviderType = "GCS"
	// ProviderTypeS3 captures enum value "S3"
	ProviderTypeS3 ProviderType = "S3"
	// ProviderTypeAzure captures enum value "Azure"
	ProviderTypeAzure ProviderType = "Azure"
)

// SecretType enum for different providers
type SecretType string

const (
	// SecretTypeAwsAccessKey captures enum value "AwsAccessKey"
	SecretTypeAwsAccessKey SecretType = "AwsAccessKey"
	// SecretTypeGcpServiceAccountKey captures enum value "GcpServiceAccountKey"
	SecretTypeGcpServiceAccountKey SecretType = "GcpServiceAccountKey"
	// SecretTypeAzStorageAccount captures enum value "AzStorageAccount"
	SecretTypeAzStorageAccount SecretType = "AzStorageAccount"
)
