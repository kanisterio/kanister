package awsefs

import (
	awsefs "github.com/aws/aws-sdk-go/service/efs"
	"github.com/kanisterio/kanister/pkg/blockstorage"
)

// bytesInGiB calculates how many GiB is equal to the given bytes by rounding up
// the intermediate result.
func bytesInGiB(bytes int64) int64 {
	const GiBInBytes int64 = int64(1024) * int64(1024) * int64(1024)
	// Round up
	return (bytes + GiBInBytes - 1) / GiBInBytes
}

// convertFromEFSTags converts AWS EFS tag structure to a flattened map.
func convertFromEFSTags(efsTags []*awsefs.Tag) map[string]string {
	tags := make(map[string]string)
	for _, t := range efsTags {
		tags[*t.Key] = *t.Value
	}
	return tags
}

// volumeFromEFSDescription converts an AWS EFS filesystem description to Kanister blockstorage Volume type
// using the information in the description.
//
// ID of the volume is equal to EFS filesystems ID (e.g fs-bdf36586).
// Iops is always set to 0.
// VolumeType and Atrributes set to corresponding empty values.
func volumeFromEFSDescription(description *awsefs.FileSystemDescription, zone string) *blockstorage.Volume {
	return &blockstorage.Volume{
		Az:           zone,
		ID:           *description.FileSystemId,
		Type:         blockstorage.TypeEFS,
		Encrypted:    *description.Encrypted,
		CreationTime: blockstorage.TimeStamp(*description.CreationTime),
		Size:         bytesInGiB(*description.SizeInBytes.Value),
		Tags:         blockstorage.MapToKeyValue(convertFromEFSTags(description.Tags)),
		Iops:         0,
		VolumeType:   "",
		Attributes:   nil,
	}
}

// volumesFromEFSDescriptions returns the list of volumes from the EFS filesystem descriptions.
func volumesFromEFSDescriptions(descriptions []*awsefs.FileSystemDescription, zone string) []*blockstorage.Volume {
	volumes := make([]*blockstorage.Volume, 0, len(descriptions))
	for _, desc := range descriptions {
		volumes = append(volumes, volumeFromEFSDescription(desc, zone))
	}
	return volumes
}
