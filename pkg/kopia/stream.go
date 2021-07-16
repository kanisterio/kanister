// Copyright 2021 The Kanister Authors.
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

package kopia

import (
	"context"
	"io"
	"path/filepath"
	"sync"

	"github.com/kopia/kopia/fs"
	"github.com/kopia/kopia/fs/virtualfs"
	"github.com/kopia/kopia/snapshot"
	"github.com/kopia/kopia/snapshot/snapshotfs"
	"github.com/pkg/errors"
)

const (
	// buffSize is default buffer size used during kopia read
	bufSize = 65536

	defaultRootDir = "/kanister-backups"
	dotDirString   = "."
	slashDirString = "/"

	pushRepoPurpose = "kando location push"
	pullRepoPurpose = "kando location pull"
)

// SnapshotInfo tracks kopia snapshot information produced by a kando command in a phase
type SnapshotInfo struct {
	// ID is the snapshot ID produced by kopia snapshot operation
	ID string `json:"id"`
	// LogicalSize is the sum of cached and hashed file size in bytes
	LogicalSize int64 `json:"logicalSize"`
	// PhysicalSize is the uploaded size in bytes
	PhysicalSize int64 `json:"physicalSize"`
}

// Validate validates SnapshotInfo field values
func (si *SnapshotInfo) Validate() error {
	if si == nil {
		return errors.New("kopia snapshotInfo cannot be nil")
	}
	if si.ID == "" {
		return errors.New("kopia snapshot ID cannot be empty")
	}
	return nil
}

// Write creates a kopia snapshot from the given reader
// A virtual directory tree rooted at filepath.Dir(path) is created with
// a kopia streaming file with filepath.Base(path) as name
func Write(ctx context.Context, path string, source io.Reader, password string) (*SnapshotInfo, error) {
	rep, err := OpenRepository(ctx, defaultConfigFilePath, password, pushRepoPurpose)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to open kopia repository")
	}

	// If the input `path` provided does not have a parent directory OR
	// has just root (`/`) directory as the parent,
	// use the default directory as root of the kopia snapshot
	parentPath := filepath.Dir(path)
	if parentPath == dotDirString || parentPath == slashDirString {
		parentPath = defaultRootDir
	}

	// Populate the source info with parent path as the source
	sourceInfo := snapshot.SourceInfo{
		UserName: rep.ClientOptions().Username,
		Host:     rep.ClientOptions().Hostname,
		Path:     parentPath,
	}

	// This creates a virtual directory tree rooted at a static directory
	// with path as `parentPath` and a kopia fs.StreamingFile as the single child entry
	rootDir := virtualfs.NewStaticDirectory(sourceInfo.Path, fs.Entries{
		virtualfs.StreamingFileFromReader(filepath.Base(path), source),
	})

	// Setup kopia uploader
	u := snapshotfs.NewUploader(rep)

	// Create a kopia snapshot
	snapID, snapshotSize, err := SnapshotSource(ctx, rep, u, sourceInfo, rootDir, "Kanister Database Backup")
	if err != nil {
		return nil, err
	}

	snapshotInfo := &SnapshotInfo{
		ID:           snapID,
		LogicalSize:  snapshotSize,
		PhysicalSize: int64(0),
	}

	return snapshotInfo, nil
}

// Read reads a kopia snapshot with the given ID and copies it to the given target
// TODO@pavan: Support files as target
func Read(ctx context.Context, backupID, path string, target io.Writer, password string) error {
	rep, err := OpenRepository(ctx, defaultConfigFilePath, password, pullRepoPurpose)
	if err != nil {
		return errors.Wrap(err, "Failed to open kopia repository")
	}

	// Get the kopia object ID belonging to the streaming file
	oid, err := getStreamingFileObjectIDFromSnapshot(ctx, rep, path, backupID)
	if err != nil {
		return err
	}

	// Open repository object and copy the data to the target
	r, err := rep.OpenObject(ctx, oid)
	if err != nil {
		return errors.Wrapf(err, "Failed to open kopia object: %v", oid)
	}

	defer r.Close() //nolint:errcheck

	_, err = copy(target, r)

	return errors.Wrap(err, "Failed to copy snapshot data to the target")
}

// bufferPool is a pool of shared buffers used during kopia read
var bufferPool = sync.Pool{
	New: func() interface{} {
		p := make([]byte, bufSize)
		return &p
	},
}

// copy is equivalent to io.Copy() but recycles the shared buffers
func copy(dst io.Writer, src io.Reader) (int64, error) {
	bufPtr := bufferPool.Get().(*[]byte)

	defer bufferPool.Put(bufPtr)

	return io.CopyBuffer(dst, src, *bufPtr)
}
