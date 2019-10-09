package vmware

import (
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

// vimID wraps ID string with vim25.ID struct.
func vimID(id string) types.ID {
	return types.ID{
		Id: id,
	}
}

func convertKeyValueToTags(kvs []types.KeyValue) []*blockstorage.KeyValue {
	tags := make(map[string]string)
	for _, kv := range kvs {
		tags[kv.Key] = kv.Value
	}
	return blockstorage.MapToKeyValue(tags)
}
