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

package blockstorage

import (
	"bytes"

	azto "github.com/Azure/azure-sdk-for-go/sdk/azcore/to"

	ktags "github.com/kanisterio/kanister/pkg/blockstorage/tags"
)

// Google Cloud environment variable names
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

// SanitizeTags are used to sanitize the tags
func SanitizeTags(tags map[string]string) map[string]string {
	// From https://cloud.google.com/compute/docs/labeling-resources
	// - Keys and values cannot be longer than 63 characters each.
	// - Keys and values can only contain lowercase letters, numeric
	//   characters, underscores, and dashes. International characters
	//   are allowed.
	// - Label keys must start with a lowercase letter and international
	//   characters are allowed.
	fixedTags := make(map[string]string)
	for k, v := range tags {
		fixedTags[ktags.SanitizeValueForGCP(k)] = ktags.SanitizeValueForGCP(v)
	}
	return fixedTags
}

// KeyValueToMap converts a KeyValue slice to a map
func KeyValueToMap(kv []*KeyValue) map[string]string {
	stringMap := make(map[string]string)
	for _, t := range kv {
		stringMap[t.Key] = t.Value
	}
	return stringMap
}

// MapToKeyValue converts a map to a KeyValue slice
func MapToKeyValue(stringMap map[string]string) []*KeyValue {
	kv := make([]*KeyValue, 0, len(stringMap))
	for k, v := range stringMap {
		kv = append(kv, &KeyValue{Key: k, Value: v})
	}
	return kv
}

// MapToString creates a string representation of a map with the entries
// separated by entrySep, and the key separated from the value using kvSep
func MapToString(m map[string]string, entrySep string, kvSep string, keyPrefix string) string {
	var b bytes.Buffer

	eSep := ""
	for k, v := range m {
		b.WriteString(eSep)
		b.WriteString(keyPrefix + k)
		b.WriteString(kvSep)
		b.WriteString(v)
		eSep = entrySep
	}
	return b.String()
}

// FilterSnapshotsWithTags filters a list of snapshots with the given tags.
func FilterSnapshotsWithTags(snapshots []*Snapshot, tags map[string]string) []*Snapshot {
	if tags == nil {
		return snapshots
	}
	result := make([]*Snapshot, 0)
	for i, snap := range snapshots {
		if ktags.IsSubset(KeyValueToMap(snap.Tags), tags) {
			result = append(result, snapshots[i])
		}
	}
	return result
}

// utility functions equivalent to old functions from package `go-autorest/autorest/to`

// StringMapPtr returns a pointer to a map of string pointers built from the passed map of strings.
func StringMapPtr(ms map[string]string) *map[string]*string {
	msp := make(map[string]*string, len(ms))
	for k, s := range ms {
		msp[k] = azto.Ptr(s)
	}
	return &msp
}

// StringMap returns a map of strings built from the map of string pointers. The empty string is
// used for nil pointers.
func StringMap(msp map[string]*string) map[string]string {
	ms := make(map[string]string, len(msp))
	for k, sp := range msp {
		if sp != nil {
			ms[k] = *sp
		} else {
			ms[k] = ""
		}
	}
	return ms
}

// StringSlice returns a string slice value for the passed string slice pointer. It returns a nil
// slice if the pointer is nil.
func StringSlice(s *[]string) []string {
	if s != nil {
		return *s
	}
	return nil
}

// SliceStringPtr returns a slice of string pointers from the passed string slice.
func SliceStringPtr(s []string) []*string {
	ms := make([]*string, len(s))
	for k, sp := range s {
		ms[k] = azto.Ptr(sp)
	}
	return ms
}

// Int returns an int value for the passed int pointer. It returns 0 if the pointer is nil.
func Int(i *int) int {
	if i != nil {
		return *i
	}
	return 0
}

// IntPtr returns a pointer to the passed int.
func IntPtr(i int) *int {
	return &i
}

// Int32 returns an int value for the passed int pointer. It returns 0 if the pointer is nil.
func Int32(i *int32) int32 {
	if i != nil {
		return *i
	}
	return 0
}

// Int32Ptr returns a pointer to the passed int32.
func Int32Ptr(i int32) *int32 {
	return &i
}

// Int64 returns an int value for the passed int pointer. It returns 0 if the pointer is nil.
func Int64(i *int64) int64 {
	if i != nil {
		return *i
	}
	return 0
}

// StringFromPtr returns a string value for the passed string pointer.
// It returns the empty string if the pointer is nil.
func StringFromPtr(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

// StringPtr returns a pointer to the passed string.
func StringPtr(s string) *string {
	return &s
}
