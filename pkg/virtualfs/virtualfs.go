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

package virtualfs

import (
	"os"
	"strings"

	"github.com/kopia/kopia/fs"
	"github.com/pkg/errors"
)

// NewDirectory returns a virtual FS root directory
func NewDirectory(rootName string) *Directory {
	if strings.Contains(rootName, "/") {
		return errors.New("Root name cannot contain '/'")
	}
	return &Directory{
		dirEntry: dirEntry{
			name: rootName,
			mode: 0777 | os.ModeDir, // nolint:gomnd
		},
	}
}

var (
	_ fs.Directory = &Directory{}
	_ fs.Entry     = &dirEntry{}
	_ fs.File      = &file{}
	_ fs.Symlink   = &inmemorySymlink{}
)
