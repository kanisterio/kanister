package awsefs

import (
	"context"
	"errors"

	"github.com/kanisterio/kanister/pkg/blockstorage"
)

type efs struct {
}

var _ blockstorage.Provider = (*efs)(nil)

// NewEFSProvider retuns a blockstorage provider for AWS EFS.
func NewEFSProvider() blockstorage.Provider {
	return &efs{}
}

func (e *efs) Type() blockstorage.Type {
	return blockstorage.TypeEFS
}

func (e *efs) VolumeCreate(context.Context, blockstorage.Volume) (*blockstorage.Volume, error) {
	return nil, errors.New("Not implemented")
}

func (e *efs) VolumeCreateFromSnapshot(ctx context.Context, snapshot blockstorage.Snapshot, tags map[string]string) (*blockstorage.Volume, error) {
	return nil, errors.New("Not implemented")
}

func (e *efs) VolumeDelete(context.Context, *blockstorage.Volume) error {
	return errors.New("Not implemented")
}

func (e *efs) VolumeGet(ctx context.Context, id string, zone string) (*blockstorage.Volume, error) {
	return nil, errors.New("Not implemented")
}

func (e *efs) SnapshotCopy(ctx context.Context, from blockstorage.Snapshot, to blockstorage.Snapshot) (*blockstorage.Snapshot, error) {
	return nil, errors.New("Not implemented")
}

func (e *efs) SnapshotCreate(ctx context.Context, volume blockstorage.Volume, tags map[string]string) (*blockstorage.Snapshot, error) {
	return nil, errors.New("Not implemented")
}

func (e *efs) SnapshotCreateWaitForCompletion(context.Context, *blockstorage.Snapshot) error {
	return errors.New("Not implemented")
}

func (e *efs) SnapshotDelete(context.Context, *blockstorage.Snapshot) error {
	return errors.New("Not implemented")
}

func (e *efs) SnapshotGet(ctx context.Context, id string) (*blockstorage.Snapshot, error) {
	return nil, errors.New("Not implemented")
}

func (e *efs) SetTags(ctx context.Context, resource interface{}, tags map[string]string) error {
	return errors.New("Not implemented")
}

func (e *efs) VolumesList(ctx context.Context, tags map[string]string, zone string) ([]*blockstorage.Volume, error) {
	return nil, errors.New("Not implemented")
}

func (e *efs) SnapshotsList(ctx context.Context, tags map[string]string) ([]*blockstorage.Snapshot, error) {
	return nil, errors.New("Not implemented")
}
