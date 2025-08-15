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

// Package volumesnapshot (tags) provides utilities for managing and manipulating tags
// that are used to label and identify resources.
package volumesnapshot

import (
	"regexp"
	"strings"
)

const (
	// ClusterTagKey is used to tag resources with the cluster name
	ClusterTagKey = "kanister.io/clustername"
	// VersionTagKey is used to tag resources with a version
	VersionTagKey = "kanister.io/version"
	// AppNameTag is used to tag volumes with the app they belong to
	AppNameTag = "kanister.io/appname"
)

// SanitizeValueForGCP shrink value if needed and change prohibited chars
func SanitizeValueForGCP(value string) string {
	// From https://cloud.google.com/compute/docs/labeling-resources
	// - Keys and values cannot be longer than 63 characters each.
	// - Keys and values can only contain lowercase letters, numeric
	//   characters, underscores, and dashes. International characters
	//   are allowed.
	// - Label keys must start with a lowercase letter and international
	//   characters are allowed.
	re := regexp.MustCompile("[^a-z0-9_-]")
	sanitizedVal := value
	if len(sanitizedVal) > 63 {
		sanitizedVal = sanitizedVal[0:63]
	}
	sanitizedVal = strings.ToLower(sanitizedVal)
	sanitizedVal = re.ReplaceAllString(sanitizedVal, "_")
	sanitizedVal = strings.TrimRight(sanitizedVal, "_-")
	return sanitizedVal
}

// IsSubset returns true if key-value pairs of 'subset' is the subset of
// key-value pairs of 'set'.
func IsSubset(set map[string]string, subset map[string]string) bool {
	for k, v := range subset {
		if v2, ok := set[k]; !ok || v != v2 {
			return false
		}
	}
	return true
}
