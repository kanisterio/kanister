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

import (
	"fmt"

	"github.com/kanisterio/kanister/pkg/logsafe"
)

const (
	prefixFlag = "--prefix"
	bucketFlag = "--bucket"
)

type StorageCommandParams struct {
	// Common params
	Location       map[string][]byte
	RepoPathPrefix string
}

// KopiaStorageArgs returns kopia command arguments for specific storage
func KopiaStorageArgs(params *StorageCommandParams) (logsafe.Cmd, error) {
	LocType := locationType(params.Location)
	switch locationType(params.Location) {
	case LocTypeFilestore:
		return filesystemArgs(params.Location, params.RepoPathPrefix), nil
	case LocTypeS3:
		return s3Args(params.Location, params.RepoPathPrefix), nil
	case LocTypeGCS:
		return gcsArgs(params.Location, params.RepoPathPrefix), nil
	case LocTypeAzure:
		return azureArgs(params.Location, params.RepoPathPrefix), nil
	case LocTypeS3Compliant:
		return s3Args(params.Location, params.RepoPathPrefix), nil
	default:
		return nil, fmt.Errorf("unsupported type for the location: %s", LocType)
	}
}
