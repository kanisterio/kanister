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

package model

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/kanisterio/safecli"

	rs "github.com/kanisterio/kanister/pkg/secrets/repositoryserver"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
)

// StorageBuilder defines a function that creates
// a safecli.Builder for the storage sub command.
type StorageBuilder func(StorageFlag) (*safecli.Builder, error)

// StorageBuilderFactory defines a factory interface
// for creating a StorageBuilder by type.
type StorageBuilderFactory interface {
	Create(rs.LocType) StorageBuilder
}

// BuildersFactory defines a map of StorageBuilder by LocType.
type BuildersFactory map[rs.LocType]StorageBuilder

// Create returns a StorageBuilder by LocType and
// implements the StorageBuilderFactory interface.
func (sb BuildersFactory) Create(locType rs.LocType) StorageBuilder {
	if b, found := sb[locType]; found {
		return b
	}
	return sb.unsupportedStorageType(locType)
}

// unsupportedStorageType returns an error for an unsupported location type.
func (sb BuildersFactory) unsupportedStorageType(locType rs.LocType) StorageBuilder {
	return func(StorageFlag) (*safecli.Builder, error) {
		return nil, errors.Wrap(cli.ErrUnsupportedStorage, fmt.Sprintf("unsupported location type: '%v'", locType))
	}
}
