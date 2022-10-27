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
	"time"

	"github.com/kanisterio/kanister/pkg/logsafe"
)

type StorageCommandParams struct {
	// S3 specific param
	AssumeRoleDuration time.Duration
	// Common params
	Location       map[string]string
	RepoPathPrefix string
}

// KopiaStorageArgs returns kopia command arguments for specific storage
func KopiaStorageArgs(params *StorageCommandParams) (logsafe.Cmd, error) {
	LocType := locationType(params.Location)
	switch locationType(params.Location) {
	case LocTypeFilestore:
		return kopiaFilesystemArgs(params.Location, params.RepoPathPrefix), nil
	case LocTypeS3:
		return kopiaS3Args(params.Location, params.AssumeRoleDuration, params.RepoPathPrefix), nil
	case LocTypeGCS:
		return kopiaGCSArgs(params.Location, params.RepoPathPrefix), nil
	case LocTypeAzure:
		return kopiaAzureArgs(params.Location, params.RepoPathPrefix), nil
	default:
		return nil, fmt.Errorf("unsupported type for the location: %s", LocType)
	}
}
