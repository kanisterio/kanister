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

package storage

type LocType string

const (
	LocTypeS3        LocType = "s3"
	LocTypeGCS       LocType = "gcs"
	LocTypeAzure     LocType = "azure"
	LocTypeFilestore LocType = "filestore"
)

const (
	bucketKey        = "bucket"
	endpointKey      = "endpoint"
	prefixKey        = "prefix"
	regionKey        = "region"
	skipSSLVerifyKey = "skipSSLVerify"
	typeKey          = "type"
)

func getBucketNameFromMap(m map[string]string) string {
	return m[bucketKey]
}

func getEndpointFromMap(m map[string]string) string {
	return m[endpointKey]
}

func getPrefixFromMap(m map[string]string) string {
	return m[prefixKey]
}

func getRegionFromMap(m map[string]string) string {
	return m[regionKey]
}

func checkSkipSSLVerifyFromMap(m map[string]string) bool {
	v := m[skipSSLVerifyKey]
	return v == "true"
}

func locationType(m map[string]string) LocType {
	return LocType(m[typeKey])
}
