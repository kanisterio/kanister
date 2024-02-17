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

package fs

import (
	"github.com/kanisterio/safecli/command"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal"
	"github.com/kanisterio/kanister/pkg/log"
)

const (
	defaultFSMountPath = "/mnt/data"
)

// New creates a new subcommand for the filesystem storage.
func New(location internal.Location, repoPathPrefix string, _ log.Logger) command.Applier {
	path, err := generateFileSystemMountPath(location.Prefix(), repoPathPrefix)
	if err != nil {
		return command.NewErrorArgument(err)
	}
	return command.NewArguments(subcmdFilesystem, optRepoPath(path))
}

// generateFileSystemMountPath generates the mount path for the filesystem storage.
func generateFileSystemMountPath(locPrefix, repoPrefix string) (string, error) {
	fullRepoPath := internal.GenerateFullRepoPath(locPrefix, repoPrefix)
	if fullRepoPath == "" {
		return "", cli.ErrInvalidRepoPath
	}
	return defaultFSMountPath + "/" + fullRepoPath, nil
}
