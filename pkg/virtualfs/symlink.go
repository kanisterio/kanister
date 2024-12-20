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
	"context"

	"github.com/kopia/kopia/fs"
)

// inmemorySymlink is a mock in-memory implementation of kopia's fs.Symlink
type inmemorySymlink struct {
	dirEntry
}

var _ fs.Symlink = (*inmemorySymlink)(nil)

func (imsl *inmemorySymlink) Readlink(ctx context.Context) (string, error) {
	panic("Symlinks not supported")
}

func (imsl *inmemorySymlink) Resolve(ctx context.Context) (fs.Entry, error) {
	panic("Resolve not supported")
}
