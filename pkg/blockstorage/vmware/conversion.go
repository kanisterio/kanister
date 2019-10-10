package vmware

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/kanisterio/kanister/pkg/blockstorage"
)

func convertFromObjectToVolume(vso *types.VStorageObject) *blockstorage.Volume {
	return &blockstorage.Volume{
		Type:         blockstorage.TypeFCD,
		ID:           vso.Config.Id.Id,
		CreationTime: blockstorage.TimeStamp(vso.Config.CreateTime),
		Size:         vso.Config.CapacityInMB / 1024,
		Az:           "",
		Iops:         0,
		Encrypted:    false,
		VolumeType:   "",
		Tags:         blockstorage.VolumeTags{},
		Attributes:   map[string]string{},
	}
}

func convertFromObjectToSnapshot(vso *types.VStorageObjectSnapshotInfoVStorageObjectSnapshot, volID string) *blockstorage.Snapshot {
	return &blockstorage.Snapshot{
		Type:         blockstorage.TypeFCD,
		CreationTime: blockstorage.TimeStamp(vso.CreateTime),
		ID:           snapshotFullID(volID, vso.Id.Id),
		Size:         0,
		Region:       "",
		Encrypted:    false,
	}
}

// vimID wraps ID string with vim25.ID struct.
func vimID(id string) types.ID {
	return types.ID{
		Id: id,
	}
}

func snapshotFullID(volID, snapshotID string) string {
	return volID + ":" + snapshotID
}

func splitSnapshotFullID(fullID string) (volID string, snapshotID string, err error) {
	split := strings.Split(fullID, ":")
	if len(split) != 2 {
		return "", "", errors.New("Malformed full ID for snapshot")
	}
	if len(split[0]) == 0 || len(split[1]) == 0 {
		return "", "", errors.New("Malformed volume ID or snapshot ID")
	}
	return split[0], split[1], nil
}

func convertKeyValueToTags(kvs []types.KeyValue) []*blockstorage.KeyValue {
	tags := make(map[string]string)
	for _, kv := range kvs {
		tags[kv.Key] = kv.Value
	}
	return blockstorage.MapToKeyValue(tags)
}
