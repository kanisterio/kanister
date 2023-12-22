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

package getter

import (
	"context"

	"github.com/pkg/errors"

	"github.com/kanisterio/kanister/pkg/blockstorage"
	"github.com/kanisterio/kanister/pkg/blockstorage/awsebs"
	"github.com/kanisterio/kanister/pkg/blockstorage/azure"
	"github.com/kanisterio/kanister/pkg/blockstorage/gcepd"
)

// Getter is a resolver for a storage provider.
type Getter interface {
	Get(blockstorage.Type, map[string]string) (blockstorage.Provider, error)
}

var _ Getter = (*getter)(nil)

type getter struct{}

// New returns a new Getter
func New() Getter {
	return &getter{}
}

// Get returns a provider for the requested storage type in the specified region
func (*getter) Get(storageType blockstorage.Type, config map[string]string) (blockstorage.Provider, error) {
	switch storageType {
	case blockstorage.TypeEBS:
		return awsebs.NewProvider(context.TODO(), config)
	case blockstorage.TypeGPD:
		return gcepd.NewProvider(config)
	case blockstorage.TypeAD:
		return azure.NewProvider(context.Background(), config)
	default:
		return nil, errors.Errorf("Unsupported storage type %v", storageType)
	}
}

// Supported returns true if the storage type is supported
func Supported(st blockstorage.Type) bool {
	switch st {
	case blockstorage.TypeEBS:
		return true
	case blockstorage.TypeGPD:
		return true
	case blockstorage.TypeAD:
		return true
	default:
		return false
	}
}
