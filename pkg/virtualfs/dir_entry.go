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
	"time"

	"github.com/kopia/kopia/fs"
)

// dirEntry is an in-memory implementation of a directory entry
type dirEntry struct {
	name    string
	mode    os.FileMode
	size    int64
	modTime time.Time
	owner   fs.OwnerInfo
}

var _ fs.Entry = (*dirEntry)(nil)

func (e dirEntry) Name() string {
	return e.name
}

func (e dirEntry) IsDir() bool {
	return e.mode.IsDir()
}

func (e dirEntry) Mode() os.FileMode {
	return e.mode
}

func (e dirEntry) ModTime() time.Time {
	return e.modTime
}

func (e dirEntry) Size() int64 {
	return e.size
}

func (e dirEntry) Sys() interface{} {
	return nil
}

func (e dirEntry) Owner() fs.OwnerInfo {
	return e.owner
}

func (e dirEntry) Device() fs.DeviceInfo {
	return fs.DeviceInfo{}
}

func (e dirEntry) LocalFilesystemPath() string {
	return ""
}

func (e dirEntry) Close() {}
