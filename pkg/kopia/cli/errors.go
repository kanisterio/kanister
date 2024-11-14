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
	"github.com/kanisterio/errkit"
)

// Common errors
var (
	// ErrInvalidID is returned when the ID is empty.
	ErrInvalidID = errkit.NewSentinelErr("invalid ID")
)

// storage errors
var (
	// ErrUnsupportedStorage is returned when the storage is not supported.
	ErrUnsupportedStorage = errkit.NewSentinelErr("unsupported storage")
	// ErrInvalidRepoPath is returned when the repoPath is empty.
	ErrInvalidRepoPath = errkit.NewSentinelErr("repository path cannot be empty")
	// ErrInvalidPrefix is returned when the prefix is empty.
	ErrInvalidPrefix = errkit.NewSentinelErr("prefix cannot be empty")
	// ErrInvalidBucketName is returned when the bucketName is empty.
	ErrInvalidBucketName = errkit.NewSentinelErr("bucket name cannot be empty")
	// ErrInvalidCredentialsFile is returned when the credentials file is empty.
	ErrInvalidCredentialsFile = errkit.NewSentinelErr("credentials file cannot be empty")
	// ErrInvalidContainerName is returned when the containerName is empty.
	ErrInvalidContainerName = errkit.NewSentinelErr("container name cannot be empty")
	// ErrInvalidServerURL is returned when the serverURL is empty.
	ErrInvalidServerURL = errkit.NewSentinelErr("server URL cannot be empty")
)
