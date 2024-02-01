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
	"github.com/pkg/errors"

	"github.com/kanisterio/safecli"

	cmdlog "github.com/kanisterio/kanister/pkg/kopia/cli/internal/log"
	log "github.com/kanisterio/kanister/pkg/log"
)

var (
	// ErrInvalidFactory is returned when the factory is nil.
	ErrInvalidFactory = errors.New("factory cannot be nil")
)

// StorageFlag is a set of flags that are used to create a StorageFlag sub command.
type StorageFlag struct {
	Location       Location
	RepoPathPrefix string

	Factory StorageBuilderFactory
	Logger  log.Logger
}

// GetLogger returns the logger.
// If the logger is nil, it returns a NopLogger.
func (s StorageFlag) GetLogger() log.Logger {
	if s.Logger == nil {
		s.Logger = &cmdlog.NopLogger{}
	}
	return s.Logger
}

// Apply applies the storage flags to the command.
func (s StorageFlag) Apply(cli safecli.CommandAppender) error {
	if s.Factory == nil {
		return ErrInvalidFactory
	}
	storageBuilder := s.Factory.Create(s.Location.Type())
	storageCLI, err := storageBuilder(s)
	if err != nil {
		return errors.Wrap(err, "failed to apply storage args")
	}
	cli.Append(storageCLI)
	return nil
}
