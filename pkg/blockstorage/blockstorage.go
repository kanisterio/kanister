// Copyright 2019 The Kanister Authors.
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

package blockstorage

import (
	"context"
)

// Provider abstracts actions on underlying storage
type Provider interface {
	// Type returns the underlying storage type
	Type() Type
	// Volume operations
	VolumeCreate(context.Context, Volume) (*Volume, error)
	VolumeCreateFromSnapshot(ctx context.Context, snapshot Snapshot, tags map[string]string) (*Volume, error)
	VolumeDelete(context.Context, *Volume) error
	VolumeGet(ctx context.Context, id string, zone string) (*Volume, error)
	// Snapshot operations
	SnapshotCopy(ctx context.Context, from Snapshot, to Snapshot) (*Snapshot, error)
	// SnapshotCopyWithArgs func is invoked to perform migration when there is a need to provide
	// additional params such as creds of target cluster to carry out the
	// Snapshot copy action, use SnapshotCopy func otherwise.
	// Currently used by Azure only.
	SnapshotCopyWithArgs(ctx context.Context, from Snapshot, to Snapshot, args map[string]string) (*Snapshot, error)
	SnapshotCreate(ctx context.Context, volume Volume, tags map[string]string) (*Snapshot, error)
	SnapshotCreateWaitForCompletion(context.Context, *Snapshot) error
	SnapshotDelete(context.Context, *Snapshot) error
	SnapshotGet(ctx context.Context, id string) (*Snapshot, error)
	// Others
	SetTags(ctx context.Context, resource interface{}, tags map[string]string) error
	VolumesList(ctx context.Context, tags map[string]string, zone string) ([]*Volume, error)
	SnapshotsList(ctx context.Context, tags map[string]string) ([]*Snapshot, error)
}

// RestoreTargeter implements the SnapshotRestoreTargets method
type RestoreTargeter interface {
	// SnapshotRestoreTargets returns whether a snapshot can be restored globally.
	// If not globally restorable, returns a map of the regions and zones to which snapshot can be restored.
	SnapshotRestoreTargets(context.Context, *Snapshot) (global bool, regionsAndZones map[string][]string, err error)
}

const SnapshotDoesNotExistError = "Snapshot does not exist"
