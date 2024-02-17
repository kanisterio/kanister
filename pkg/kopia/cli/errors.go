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

package cli

import (
	"github.com/pkg/errors"
)

// Common errors
var (
	// ErrInvalidID is returned when the ID is empty.
	ErrInvalidID = errors.New("invalid ID")
)

// storage errors
var (
	// ErrUnsupportedStorage is returned when the storage is not supported.
	ErrUnsupportedStorage = errors.New("unsupported storage")
	// ErrInvalidRepoPath is returned when the repoPath is empty.
	ErrInvalidRepoPath = errors.New("repository path cannot be empty")
)