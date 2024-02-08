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

package storage

import (
	"context"

	kopiaerrors "github.com/kanisterio/kanister/pkg/kopia/errors"
	"github.com/kopia/kopia/repo/blob"
	"github.com/pkg/errors"
)

type StorageType string

const (
	TypeS3        StorageType = "S3"
	TypeAzure     StorageType = "Azure"
	TypeFileStore StorageType = "FileStore"
	TypeGCP       StorageType = "GCP"
)

type Storage interface {
	New() (blob.Storage, error)
	SetOptions(context.Context, map[string]string) error
}

func New(storageType StorageType) Storage {
	switch storageType {
	case TypeS3:
		return &s3Storage{}
	default:
		return nil
	}
}

func validateCommonStorageArgs(options map[string]string, storageType StorageType, requiredArgs []string) error {
	if options == nil {
		return errors.Errorf(kopiaerrors.ErrStorageOptionsCannotBeNilMsg, string(storageType))
	}
	for _, arg := range requiredArgs {
		if _, ok := options[arg]; !ok {
			return errors.Errorf(kopiaerrors.ErrMissingRequiredFieldMsg, arg, string(storageType))
		}
	}
	return nil
}
