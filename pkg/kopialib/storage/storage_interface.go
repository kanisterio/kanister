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

package storage

import "github.com/kopia/kopia/repo/blob"

type StorageType string

const (
	TypeS3        StorageType = "S3"
	TypeAzure     StorageType = "Azure"
	TypeFileStore StorageType = "fileStore"
)

type storage interface {
	GetStorage() (blob.Storage, error)
}

func New(storageType StorageType) storage {
	switch storageType {
	case TypeS3:
		return s3Storage{}
	case TypeFileStore:
		return fileSystem{}
	case TypeAzure:
		return azureStorage{}
	default:
		return nil
	}
}
