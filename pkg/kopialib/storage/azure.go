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

	"github.com/kanisterio/kanister/pkg/kopialib"
	"github.com/kopia/kopia/repo/blob"
	"github.com/kopia/kopia/repo/blob/azure"
)

type azureStorage struct {
	Options *azure.Options
	Create  bool
}

func (a azureStorage) Connect() (blob.Storage, error) {
	return azure.New(context.Background(), a.Options, a.Create)
}

func (a *azureStorage) WithOptions(opts azure.Options) {
	a.Options = &opts
}

func (a *azureStorage) WithCreate(create bool) {
	a.Create = create
}

func (a *azureStorage) SetOptions(ctx context.Context, options map[string]string) {
	a.Options = &azure.Options{
		Prefix:         options[kopialib.PrefixKey],
		Container:      options[kopialib.BucketKey],
		StorageAccount: options[kopialib.AzureStorageAccount],
		StorageKey:     options[kopialib.AzureStorageAccountAccessKey],
		SASToken:       options[kopialib.AzureStorageAccountAccessKey],
	}
}
