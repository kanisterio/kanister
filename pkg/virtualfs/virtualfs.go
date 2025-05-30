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

// Package virtualfs provides helper functions for managing file system.
package virtualfs

import (
	"os"
	"strings"

	"github.com/kanisterio/errkit"
)

// NewDirectory returns a virtual FS root directory
func NewDirectory(rootName string) (*Directory, error) {
	if strings.Contains(rootName, "/") {
		return nil, errkit.New("Root name cannot contain '/'")
	}
	return &Directory{
		dirEntry: dirEntry{
			name: rootName,
			mode: 0777 | os.ModeDir, //nolint:gomnd
		},
	}, nil
}
