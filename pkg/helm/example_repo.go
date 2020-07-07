// Copyright 2020 The Kanister Authors.
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

package helm

import (
	"os"
	"path/filepath"
	"runtime"
)

// KanisterExamplesRepoPath attempts to locate the path for the Kanister helm
// examples repo.
//
// An error is returned if the repo doesn't exist in the local filesystem. This
// happens when the source is unavailable on the local system.
func KanisterExamplesRepoPath() (string, error) {
	_, goSource, _, _ := runtime.Caller(0)

	repoPath := filepath.Join(filepath.Dir(goSource), "..", "..", "examples", "helm", "kanister")
	_, err := os.Stat(repoPath)
	if os.IsNotExist(err) {
		return "", os.ErrNotExist
	}

	return repoPath, nil
}
