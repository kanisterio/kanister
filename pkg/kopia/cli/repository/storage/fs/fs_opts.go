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
)

var (
	subcmdFilesystem = command.NewArgument("filesystem")
)

// optRepoPath creates a new path option with a given path.
// If the path is empty, it returns an error.
func optRepoPath(path string) command.Applier {
	if path == "" {
		return command.NewErrorArgument(cli.ErrInvalidRepoPath)
	}
	return command.NewOptionWithArgument("--path", path)
}
