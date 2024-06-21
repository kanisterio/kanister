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

package internal

import (
	"strconv"
	"strings"

	rs "github.com/kanisterio/kanister/pkg/secrets/repositoryserver"
)

// Location is a map of key-value pairs that represent different storage properties.
type Location map[string][]byte

// Type returns the location type.
func (l Location) Type() rs.LocType {
	return rs.LocType(string(l[rs.TypeKey]))
}

// Region returns the location region.
func (l Location) Region() string {
	return string(l[rs.RegionKey])
}

// BucketName returns the location bucket name.
func (l Location) BucketName() string {
	return string(l[rs.BucketKey])
}

// Endpoint returns the location endpoint.
func (l Location) Endpoint() string {
	return string(l[rs.EndpointKey])
}

// Prefix returns the location prefix.
func (l Location) Prefix() string {
	return string(l[rs.PrefixKey])
}

// IsInsecureEndpoint returns true if the location endpoint is insecure/http.
func (l Location) IsInsecureEndpoint() bool {
	return strings.HasPrefix(l.Endpoint(), "http:")
}

// HasSkipSSLVerify returns true if the location has skip SSL verification.
func (l Location) HasSkipSSLVerify() bool {
	v, _ := strconv.ParseBool(string(l[rs.SkipSSLVerifyKey]))
	return v
}

// IsPointInTypeSupported returns true if the location supports point-in-time recovery.
// Currently, only S3 and Azure support point-in-time recovery.
func (l Location) IsPointInTypeSupported() bool {
	switch l.Type() {
	case rs.LocTypeAzure, rs.LocTypeS3, rs.LocTypes3Compliant:
		return true
	default:
		return false
	}
}
