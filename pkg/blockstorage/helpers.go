package blockstorage

import (
	"bytes"

	ktags "github.com/kanisterio/kanister/pkg/blockstorage/tags"
)

// Google Cloud environment variable names
const (
	GoogleCloudZone     = "CLOUDSDK_COMPUTE_ZONE"
	GoogleCloudCreds    = "GOOGLE_APPLICATION_CREDENTIALS"
	GoogleProjectID     = "projectID"
	GoogleServiceKey    = "serviceKey"
	AzureStorageAccount = "AZURE_STORAGE_ACCOUNT_NAME"
	AzureStorageKey     = "AZURE_STORAGE_ACCOUNT_KEY"
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
func MapToString(m map[string]string, entrySep string, kvSep string) string {
	var b bytes.Buffer

	eSep := ""
	for k, v := range m {
		b.WriteString(eSep)
		b.WriteString(k)
		b.WriteString(kvSep)
		b.WriteString(v)
		eSep = entrySep
	}
	return b.String()
}
