// Copyright 2022 The Kanister Authors.
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

package internal

import (
	"path"
)

// GenerateFullRepoPath generates the full repository path.
// If the location-specific prefix is empty, the repository-specific prefix is returned.
func GenerateFullRepoPath(locPrefix, repoPathPrefix string) string {
	if locPrefix != "" {
		return path.Join(locPrefix, repoPathPrefix) + "/"
	}
	return repoPathPrefix
}
