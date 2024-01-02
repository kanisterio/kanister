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

package mockblockstorage

import (
	"context"
	"fmt"
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"github.com/kanisterio/kanister/pkg/blockstorage"
	"github.com/kanisterio/kanister/pkg/blockstorage/getter"
)

var _ blockstorage.Provider = (*Provider)(nil)

// Provider implements a mock storage provider
type Provider struct {
	storageType       blockstorage.Type
	volume            blockstorage.Volume
	snapshot          blockstorage.Snapshot
	failPoints        map[string]error
	SnapIDList        []string
	DeletedSnapIDList []string
	VolIDList         []string
}

var _ getter.Getter = (*mockGetter)(nil)

type mockGetter struct{}

// NewGetter returns a new mockGetter
func NewGetter() getter.Getter {
	return &mockGetter{}
}

// Get returns a provider for the requested storage type in the specified region
func (*mockGetter) Get(storageType blockstorage.Type, config map[string]string) (blockstorage.Provider, error) {
	// TODO(tom): we might want to honor these settings.
	switch storageType {
	case blockstorage.TypeEBS:
		fallthrough
	case blockstorage.TypeGPD:
		return Get(storageType)
	default:
		return nil, errors.New("Get failed")
	}
}

// Get returns a mock storage provider
func Get(storageType blockstorage.Type) (*Provider, error) {
	volumeUUID, err := uuid.NewV1()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create UUID")
	}
	volume := blockstorage.Volume{
		Type:        storageType,
		ID:          fmt.Sprintf("vol-%s", volumeUUID.String()),
		Az:          "AZ",
		Encrypted:   false,
		VolumeType:  "ssd",
		SizeInBytes: 1024,
		Iops:        0,
		Tags: []*blockstorage.KeyValue{
			{Key: "kanister.io/jobid", Value: "unittest"},
			{Key: "kanister.io/volid", Value: "vol"},
		},
		CreationTime: blockstorage.TimeStamp(time.Time{}),
	}
	snapVol := volume
	snapUUID, err := uuid.NewV1()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create UUID")
	}
	snapshot := blockstorage.Snapshot{
		Type:        storageType,
		ID:          fmt.Sprintf("snap-%s", snapUUID.String()),
		SizeInBytes: 1024,
		Tags: []*blockstorage.KeyValue{
			{Key: "kanister.io/jobid", Value: "unittest"},
			{Key: "kanister.io/snapid", Value: "snap"},
		},
		Volume:       &snapVol,
		CreationTime: blockstorage.TimeStamp(time.Time{}),
	}
	return &Provider{
		storageType:       storageType,
		volume:            volume,
		snapshot:          snapshot,
		failPoints:        make(map[string]error),
		SnapIDList:        make([]string, 0),
		DeletedSnapIDList: make([]string, 0),
		VolIDList:         make([]string, 0),
	}, nil
}

// Type mock
func (p *Provider) Type() blockstorage.Type {
	return p.storageType
}

// VolumeCreate mock
func (p *Provider) VolumeCreate(context.Context, blockstorage.Volume) (*blockstorage.Volume, error) {
	return p.MockVolume(), nil
}

// VolumeCreateFromSnapshot mock
func (p *Provider) VolumeCreateFromSnapshot(ctx context.Context, snapshot blockstorage.Snapshot, tags map[string]string) (*blockstorage.Volume, error) {
	volUUID, err := uuid.NewV1()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create UUID")
	}
	vol := blockstorage.Volume{
		Type:        snapshot.Type,
		ID:          fmt.Sprintf("vol-%s", volUUID.String()),
		Az:          "AZ",
		Encrypted:   false,
		VolumeType:  "ssd",
		SizeInBytes: 1024,
		Iops:        0,
		Tags: []*blockstorage.KeyValue{
			{Key: "kanister.io/jobid", Value: "unittest"},
			{Key: "kanister.io/volid", Value: "vol"},
		},
		CreationTime: blockstorage.TimeStamp(time.Time{}),
	}
	p.AddVolID(vol.ID)
	return &vol, nil
}

// VolumeDelete mock
func (p *Provider) VolumeDelete(context.Context, *blockstorage.Volume) error {
	return nil
}

// VolumeGet mock
func (p *Provider) VolumeGet(ctx context.Context, id string, zone string) (*blockstorage.Volume, error) {
	if err := p.checkFailPoint(id); err != nil {
		return nil, err
	}
	return p.MockVolume(), nil
}

// SnapshotCopy mock
func (p *Provider) SnapshotCopy(ctx context.Context, from, to blockstorage.Snapshot) (*blockstorage.Snapshot, error) {
	return p.MockSnapshot(), nil
}

// SnapshotCopyWithArgs mock
func (p *Provider) SnapshotCopyWithArgs(ctx context.Context, from, to blockstorage.Snapshot, args map[string]string) (*blockstorage.Snapshot, error) {
	return p.MockSnapshot(), nil
}

// SnapshotCreate mock
func (p *Provider) SnapshotCreate(ctx context.Context, volume blockstorage.Volume, tags map[string]string) (*blockstorage.Snapshot, error) {
	return p.MockSnapshot(), nil
}

// SnapshotCreateWaitForCompletion mock
func (p *Provider) SnapshotCreateWaitForCompletion(context.Context, *blockstorage.Snapshot) error {
	return nil
}

// SnapshotDelete mock
func (p *Provider) SnapshotDelete(ctx context.Context, snapshot *blockstorage.Snapshot) error {
	p.AddDeletedSnapID(snapshot.ID)
	return nil
}

// SnapshotGet mock
func (p *Provider) SnapshotGet(ctx context.Context, id string) (*blockstorage.Snapshot, error) {
	ret := p.snapshot
	ret.ID = id
	p.AddSnapID(id)
	return &ret, nil
}

// SetTags mock
func (p *Provider) SetTags(ctx context.Context, resource interface{}, tags map[string]string) error {
	switch res := resource.(type) {
	case *blockstorage.Volume:
		return nil
	case *blockstorage.Snapshot:
		return nil
	default:
		return errors.Errorf("Unsupported resource type %v(%T)", res, res)
	}
}

// MockVolume returns the mock volume used in the provider
func (p *Provider) MockVolume() *blockstorage.Volume {
	ret := p.volume
	return &ret
}

// MockSnapshot returns the mock snapshot used in the provider
func (p *Provider) MockSnapshot() *blockstorage.Snapshot {
	ret := p.snapshot
	return &ret
}

// VolumesList mock
func (p *Provider) VolumesList(ctx context.Context, tags map[string]string, zone string) ([]*blockstorage.Volume, error) {
	return []*blockstorage.Volume{p.MockVolume(), p.MockVolume()}, nil
}

// SnapshotsList mock
func (p *Provider) SnapshotsList(ctx context.Context, tags map[string]string) ([]*blockstorage.Snapshot, error) {
	return []*blockstorage.Snapshot{p.MockSnapshot(), p.MockSnapshot()}, nil
}

// InjectFailure adds an id that provider operations should be failed on
func (p *Provider) InjectFailure(id string, err error) {
	p.failPoints[id] = err
}

func (p *Provider) checkFailPoint(id string) error {
	if err, ok := p.failPoints[id]; ok {
		return err
	}
	return nil
}

// AddSnapID adds id to the list of snapshot id's
func (p *Provider) AddSnapID(id string) {
	if present := CheckID(id, p.SnapIDList); !present {
		p.SnapIDList = append(p.SnapIDList, id)
	}
}

// AddDeletedSnapID adds id to the list of delted snapshot id's
func (p *Provider) AddDeletedSnapID(id string) {
	if present := CheckID(id, p.DeletedSnapIDList); !present {
		p.DeletedSnapIDList = append(p.DeletedSnapIDList, id)
	}
}

// AddVolID adds id to the list of volume id's
func (p *Provider) AddVolID(id string) {
	if present := CheckID(id, p.VolIDList); !present {
		p.VolIDList = append(p.VolIDList, id)
	}
}

// CheckID checks if the id is present in the list
func CheckID(id string, list []string) bool {
	for _, i := range list {
		if i == id {
			return true
		}
	}
	return false
}
