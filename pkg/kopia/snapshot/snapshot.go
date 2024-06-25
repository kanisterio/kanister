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

package snapshot

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kanisterio/errkit"
	"github.com/kopia/kopia/fs"
	"github.com/kopia/kopia/repo"
	"github.com/kopia/kopia/repo/manifest"
	"github.com/kopia/kopia/snapshot"
	"github.com/kopia/kopia/snapshot/policy"
	"github.com/kopia/kopia/snapshot/snapshotfs"

	"github.com/kanisterio/kanister/pkg/kopia"
	"github.com/kanisterio/kanister/pkg/kopia/repository"
)

// SnapshotSource creates and uploads a kopia snapshot to the given repository
func SnapshotSource(
	ctx context.Context,
	rep repo.RepositoryWriter,
	u *snapshotfs.Uploader,
	sourceInfo snapshot.SourceInfo,
	rootDir fs.Entry,
	description string,
) (string, int64, error) {
	fmt.Printf("Snapshotting %v ...\n", sourceInfo)

	snapshotStartTime := time.Now()

	previous, err := findPreviousSnapshotManifest(ctx, rep, sourceInfo, nil)
	if err != nil {
		return "", 0, errkit.Wrap(err, "Failed to find previous kopia manifests")
	}

	policyTree, err := policy.TreeForSource(ctx, rep, sourceInfo)
	if err != nil {
		return "", 0, errkit.Wrap(err, "Failed to get kopia policy tree")
	}

	manifest, err := u.Upload(ctx, rootDir, policyTree, sourceInfo, previous...)
	if err != nil {
		return "", 0, errkit.Wrap(err, "Failed to upload the kopia snapshot")
	}

	manifest.Description = description

	if _, err := snapshot.SaveSnapshot(ctx, rep, manifest); err != nil {
		return "", 0, errkit.Wrap(err, "Failed to save kopia manifest")
	}

	// TODO: https://github.com/kanisterio/kanister/issues/2441
	// _, err = policy.ApplyRetentionPolicy(ctx, rep, sourceInfo, true)
	// if err != nil {
	// 	return "", 0, errkit.Wrap(err, "Failed to apply kopia retention policy")
	// }

	if err = policy.SetManual(ctx, rep, sourceInfo); err != nil {
		return "", 0, errkit.Wrap(err, "Failed to set manual field in kopia scheduling policy for source")
	}

	if ferr := rep.Flush(ctx); ferr != nil {
		return "", 0, errkit.Wrap(ferr, "Failed to flush kopia repository")
	}

	return reportStatus(ctx, snapshotStartTime, manifest)
}

func reportStatus(ctx context.Context, snapshotStartTime time.Time, manifest *snapshot.Manifest) (string, int64, error) {
	manifestID := manifest.ID
	snapSize := manifest.Stats.TotalFileSize

	fmt.Printf("\nCreated snapshot with root %v and ID %v in %v\n", manifest.RootObjectID(), manifestID, time.Since(snapshotStartTime).Truncate(time.Second))

	// Even if the manifest is created, it might contain fatal errors and failed entries
	var errs []string
	if ds := manifest.RootEntry.DirSummary; ds != nil {
		for _, ent := range ds.FailedEntries {
			errs = append(errs, ent.Error)
		}
	}
	if len(errs) != 0 {
		return "", 0, errkit.New(strings.Join(errs, "\n"))
	}

	return string(manifestID), snapSize, nil
}

// Delete deletes Kopia snapshot with given manifest ID
func Delete(ctx context.Context, backupID, path, password string) error {
	rep, err := repository.Open(ctx, kopia.DefaultClientConfigFilePath, password, pullRepoPurpose)
	if err != nil {
		return errkit.Wrap(err, "Failed to open kopia repository")
	}

	// Load the kopia snapshot with the given backupID
	m, err := snapshot.LoadSnapshot(ctx, rep, manifest.ID(backupID))
	if err != nil {
		return errkit.Wrap(err, "Failed to load kopia snapshot with ID", "backupId", backupID)
	}
	if err := rep.DeleteManifest(ctx, m.ID); err != nil {
		return err
	}
	return rep.Flush(ctx)
}

// findPreviousSnapshotManifest returns the list of previous snapshots for a given source,
// including last complete snapshot
func findPreviousSnapshotManifest(ctx context.Context, rep repo.Repository, sourceInfo snapshot.SourceInfo, noLaterThan *fs.UTCTimestamp) ([]*snapshot.Manifest, error) {
	man, err := snapshot.ListSnapshots(ctx, rep, sourceInfo)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to list previous kopia snapshots")
	}

	// find latest complete snapshot
	var previousComplete *snapshot.Manifest
	var result []*snapshot.Manifest

	for _, p := range man {
		if noLaterThan != nil && p.StartTime.After(*noLaterThan) {
			continue
		}

		if p.IncompleteReason == "" && (previousComplete == nil || p.StartTime.After(previousComplete.StartTime)) {
			previousComplete = p
		}
	}

	if previousComplete != nil {
		result = append(result, previousComplete)
	}

	return result, nil
}

// MarshalKopiaSnapshot encodes kopia SnapshotInfo struct into a string
func MarshalKopiaSnapshot(snapInfo *SnapshotInfo) (string, error) {
	if err := snapInfo.Validate(); err != nil {
		return "", err
	}
	snap, err := json.Marshal(snapInfo)
	if err != nil {
		return "", errkit.Wrap(err, "failed to marshal kopia snapshot information")
	}

	return string(snap), nil
}

// UnmarshalKopiaSnapshot decodes a kopia snapshot JSON string into SnapshotInfo struct
func UnmarshalKopiaSnapshot(snapInfoJSON string) (SnapshotInfo, error) {
	snap := SnapshotInfo{}
	if err := json.Unmarshal([]byte(snapInfoJSON), &snap); err != nil {
		return snap, errkit.Wrap(err, "failed to unmarshal kopia snapshot information")
	}
	return snap, snap.Validate()
}
