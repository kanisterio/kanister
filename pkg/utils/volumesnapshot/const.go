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

package volumesnapshot

// Type is the type of storage supported
type Type string

const (
	// TypeAD captures enum value "AD"
	TypeAD Type = "AD"
	// TypeEBS captures enum value "EBS"
	TypeEBS Type = "EBS"
	// TypeGPD captures enum value "GPD"
	TypeGPD Type = "GPD"
	// TypeCinder captures enum value "Cinder"
	TypeCinder Type = "Cinder"
	// TypeGeneric captures enum value "Generic"
	TypeGeneric Type = "Generic"
	// TypeCeph captures enum value "Ceph"
	TypeCeph Type = "Ceph"
	// TypeEFS captures enum value "EFS"
	TypeEFS Type = "EFS"
	// TypeFCD capture enum value for "VMWare FCD"
	TypeFCD Type = "FCD"
)

// Cloud environment variable names
const (
	GoogleCloudZone            = "CLOUDSDK_COMPUTE_ZONE"
	GoogleCloudCreds           = "GOOGLE_APPLICATION_CREDENTIALS"
	GoogleProjectID            = "projectID"
	GoogleServiceKey           = "serviceKey"
	AzureStorageAccount        = "AZURE_STORAGE_ACCOUNT_NAME"
	AzureStorageKey            = "AZURE_STORAGE_ACCOUNT_KEY"
	AzureSubscriptionID        = "AZURE_SUBSCRIPTION_ID"
	AzureTenantID              = "AZURE_TENANT_ID"
	AzureClientID              = "AZURE_CLIENT_ID"
	AzureClientSecret          = "AZURE_CLIENT_SECRET"
	AzureResurceGroup          = "AZURE_RESOURCE_GROUP"
	AzureResurceMgrEndpoint    = "AZURE_RESOURCE_MANAGER_ENDPOINT"
	AzureMigrateStorageAccount = "AZURE_MIGRATE_STORAGE_ACCOUNT_NAME"
	AzureMigrateStorageKey     = "AZURE_MIGRATE_STORAGE_ACCOUNT_KEY"
	AzureMigrateResourceGroup  = "AZURE_MIGRATE_RESOURCE_GROUP"
	AzureActiveDirEndpoint     = "AZURE_AD_ENDPOINT"
	AzureActiveDirResourceID   = "AZURE_AD_RESOURCE"
	AzureCloudEnvironmentID    = "AZURE_CLOUD_ENV_ID"
)

// Error messages
const (
	SnapshotDoesNotExistError Type = "Snapshot does not exist"
)
