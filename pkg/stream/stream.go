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

package stream

import (
	"context"
	"os"
	"path/filepath"

	"github.com/kopia/kopia/snapshot"
	"github.com/kopia/kopia/snapshot/snapshotfs"
	"github.com/pkg/errors"

	"github.com/kanisterio/kanister/pkg/kopia/repository"
	kansnapshot "github.com/kanisterio/kanister/pkg/kopia/snapshot"
	"github.com/kanisterio/kanister/pkg/virtualfs"
)

const (
	snapshotDescription             = "Snapshot created by kando stream push"
	defaultPermissions  os.FileMode = 0777
)

// Push streams data to object store by reading it from the given endpoint into an in-memory filesystem
func Push(ctx context.Context, configFile, dirPath, filePath, password, sourceEndpoint string) error {
	rep, err := repository.Open(ctx, configFile, password, "kanister stream push")
	if err != nil {
		return errors.Wrap(err, "Failed to open kopia repository")
	}
	// Initialize a directory tree with given file
	// The following will create <dirPath>/<filePath> objects
	// Example: If dirPath is `/mnt/data` and filePath is `dir/file`,
	// `data` will be the root directory and
	// `dir/file` objects will be created under it
	root, err := virtualfs.NewDirectory(filepath.Base(dirPath))
	if err != nil {
		return errors.Wrap(err, "Failed to create root directory")
	}
	if _, err = virtualfs.AddFileWithStreamSource(root, filePath, sourceEndpoint, defaultPermissions, defaultPermissions); err != nil {
		return errors.Wrap(err, "Failed to add file with the given stream source to the root directory")
	}

	// Setup kopia uploader
	u := snapshotfs.NewUploader(rep)
	// Fail full snapshot if errors are encountered during upload
	u.FailFast = true

	// Populate the source info with source path and file
	sourceInfo := snapshot.SourceInfo{
		UserName: rep.ClientOptions().Username,
		Host:     rep.ClientOptions().Hostname,
		Path:     dirPath,
	}

	// Create a kopia snapshot
	_, _, err = kansnapshot.SnapshotSource(ctx, rep, u, sourceInfo, root, snapshotDescription)
	return errors.Wrap(err, "Failed to create kopia snapshot")
}
