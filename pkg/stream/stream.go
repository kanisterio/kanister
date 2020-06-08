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
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kopia/kopia/fs"
	"github.com/kopia/kopia/repo"
	"github.com/kopia/kopia/snapshot"
	"github.com/kopia/kopia/snapshot/policy"
	"github.com/kopia/kopia/snapshot/snapshotfs"
	"github.com/pkg/errors"

	"github.com/kanisterio/kanister/pkg/virtualfs"
)

const (
	snapshotDescription = "Snapshot created by kando stream push"
)

// Push streams data to object store by reading it from the given endpoint into an in-memory filesystem
func Push(ctx context.Context, dirPath, file, password, sourceEndpoint string) error {
	rep, err := OpenKopiaRepository(ctx, password)
	if err != nil {
		return errors.Wrap(err, "Failed to open kopia repository")
	}
	// Initialize a directory tree with given file
	// The following will create /dirPath/<file>
	// Example: If dirPath is `/mnt/data` and file is `/dir/file`,
	// the virtualfs creates `/mnt/data/dir/file` objects
	root := virtualfs.NewDirectory(filepath.Base(dirPath))
	if _, err = root.AddFileWithStreamSource(file, sourceEndpoint, 0777); err != nil {
		return err
	}

	// Setup kopia uploader
	u := snapshotfs.NewUploader(rep)

	// Populate the source info with source path and file
	sourceInfo := snapshot.SourceInfo{
		UserName: rep.Username(),
		Host:     rep.Hostname(),
		Path:     dirPath,
	}

	// Create a kopia snapshot
	return SnapshotSource(ctx, rep, u, sourceInfo, root)
}

// OpenKopiaRepository connects to the kopia repository based on the config
// NOTE: This assumes that `kopia repository connect` has been already run on the machine
func OpenKopiaRepository(ctx context.Context, password string) (repo.Repository, error) {
	if _, err := os.Stat(defaultConfigFileName()); os.IsNotExist(err) {
		return nil, errors.New("Failed find kopia configuration file")
	}

	r, err := repo.Open(ctx, defaultConfigFileName(), password, &repo.Options{})
	if os.IsNotExist(err) {
		return nil, errors.New("Failed to find kopia repository, use `kopia repository connect`")
	}

	return r, errors.Wrap(err, "Failed to open kopia repository")
}

func defaultConfigFileName() string {
	return filepath.Join(os.Getenv("HOME"), ".config", "kopia", "repository.config")
}

// SnapshotSource creates and uploads a kopia snapshot to the given repository
func SnapshotSource(ctx context.Context, rep repo.Repository, u *snapshotfs.Uploader, sourceInfo snapshot.SourceInfo, rootDir fs.Entry) error {
	fmt.Printf("Snapshotting %v ...\n", sourceInfo)

	t0 := time.Now()

	previous, err := findPreviousSnapshotManifest(ctx, rep, sourceInfo, nil)
	if err != nil {
		return errors.Wrap(err, "Failed to find previous kopia manifests")
	}

	policyTree, err := policy.TreeForSource(ctx, rep, sourceInfo)
	if err != nil {
		return errors.Wrap(err, "Failed to get kopia policy tree")
	}

	manifest, err := u.Upload(ctx, rootDir, policyTree, sourceInfo, previous...)
	if err != nil {
		return errors.Wrap(err, "Failed to upload the kopia snapshot")
	}

	manifest.Description = snapshotDescription

	snapID, err := snapshot.SaveSnapshot(ctx, rep, manifest)
	if err != nil {
		return errors.Wrap(err, "Failed to save kopia manifest")
	}

	_, err = policy.ApplyRetentionPolicy(ctx, rep, sourceInfo, true)
	if err != nil {
		return errors.Wrap(err, "Failed to apply kopia retention policy")
	}

	if ferr := rep.Flush(ctx); ferr != nil {
		return errors.Wrap(ferr, "Failed to flush kopia repository")
	}

	var maybePartial string
	if manifest.IncompleteReason != "" {
		maybePartial = " partial"
	}

	fmt.Printf("\nCreated%v snapshot with root %v and ID %v in %v\n", maybePartial, manifest.RootObjectID(), snapID, time.Since(t0).Truncate(time.Second))

	return err
}

// findPreviousSnapshotManifest returns the list of previous snapshots for a given source, including
// last complete snapshot and possibly some number of incomplete snapshots following it.
func findPreviousSnapshotManifest(ctx context.Context, rep repo.Repository, sourceInfo snapshot.SourceInfo, noLaterThan *time.Time) ([]*snapshot.Manifest, error) {
	man, err := snapshot.ListSnapshots(ctx, rep, sourceInfo)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to list previous kopia snapshots")
	}

	// Phase 1 - find latest complete snapshot.
	var previousComplete *snapshot.Manifest
	var previousCompleteStartTime time.Time
	var result []*snapshot.Manifest

	for _, p := range man {
		if noLaterThan != nil && p.StartTime.After(*noLaterThan) {
			continue
		}

		if p.IncompleteReason == "" && (previousComplete == nil || p.StartTime.After(previousComplete.StartTime)) {
			previousComplete = p
			previousCompleteStartTime = p.StartTime
		}
	}

	if previousComplete != nil {
		result = append(result, previousComplete)
	}

	// Add all incomplete snapshots after that
	for _, p := range man {
		if noLaterThan != nil && p.StartTime.After(*noLaterThan) {
			continue
		}

		if p.IncompleteReason != "" && p.StartTime.After(previousCompleteStartTime) {
			result = append(result, p)
		}
	}

	return result, nil
}
