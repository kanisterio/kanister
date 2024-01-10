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

package storage

import (
	"context"
	"path/filepath"

	"github.com/kanisterio/kanister/pkg/kopialib"
	"github.com/kopia/kopia/repo/blob"
	"github.com/kopia/kopia/repo/blob/filesystem"
)

const (
	defaultFileMode = 0o600
	defaultDirMode  = 0o700
)

type fileSystem struct {
	Options *filesystem.Options
	Create  bool
}

func (f *fileSystem) Connect() (blob.Storage, error) {
	return filesystem.New(context.Background(), f.Options, f.Create)
}

func (f *fileSystem) WithOptions(opts filesystem.Options) {
	f.Options = &opts
}

func (f *fileSystem) WithCreate(create bool) {
	f.Create = create
}

func (f *fileSystem) SetOptions(ctx context.Context, options map[string]string) {
	filePath := filepath.Join(kopialib.DefaultFSMountPath, options[kopialib.FilesystorePath])
	f.Options = &filesystem.Options{
		Path:          filePath,
		FileMode:      defaultFileMode,
		DirectoryMode: defaultDirMode,
	}
}
