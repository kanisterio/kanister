package awsefs

import (
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	awsarn "github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/backup"
	awsefs "github.com/aws/aws-sdk-go/service/efs"
	"github.com/pkg/errors"

	"github.com/kanisterio/kanister/pkg/blockstorage"
)

const (
	securityGroupSeperator = "+"
	mountTargetKeyPrefix   = "kasten.io/aws-mount-target/"
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

// efsIDFromResourceARN gets EFS filesystem ID from an EFS resource ARN.
func efsIDFromResourceARN(arn string) (string, error) {
	resourceARN, err := awsarn.Parse(arn)
	if err != nil {
		return "", errors.Wrap(err, "Failed to parse ARN")
	}
	// An example of resourceArn.Resource:
	// "file-system/fs-87b1302c"
	tokens := strings.Split(resourceARN.Resource, "/")
	if len(tokens) != 2 {
		return "", errors.New("Bad resource in ARN")
	}
	if tokens[0] != "file-system" {
		return "", errors.New("Bad resource type in ARN")
	}
	return tokens[1], nil
}

func snapshotFromRecoveryPoint(rp *backup.DescribeRecoveryPointOutput, volume *blockstorage.Volume, region string) (*blockstorage.Snapshot, error) {
	if rp.CreationDate == nil {
		return nil, errors.New("Recovery point has no CreationDate")
	}
	if rp.BackupSizeInBytes == nil {
		return nil, errors.New("Recovery point has no BackupSizeInBytes")
	}
	if rp.RecoveryPointArn == nil {
		return nil, errors.New("Recovery point has no RecoveryPointArn")
	}
	return &blockstorage.Snapshot{
		ID:           *rp.RecoveryPointArn,
		CreationTime: blockstorage.TimeStamp(*rp.CreationDate),
		Size:         bytesInGiB(*rp.BackupSizeInBytes),
		Region:       region,
		Type:         blockstorage.TypeEFS,
		Volume:       volume,
		Encrypted:    volume.Encrypted,
		Tags:         nil,
	}, nil
}

// convertToBackupTags converts a flattened map to AWS Backup compliant tag structure.
func convertToBackupTags(tags map[string]string) map[string]*string {
	backupTags := make(map[string]*string)
	for k, v := range tags {
		vPtr := new(string)
		*vPtr = v
		backupTags[k] = vPtr
	}
	return backupTags
}

// convertToEFSTags converts a flattened map to AWS EFS tag structure.
func convertToEFSTags(tags map[string]string) []*awsefs.Tag {
	efsTags := make([]*awsefs.Tag, 0, len(tags))
	for k, v := range tags {
		efsTags = append(efsTags, &awsefs.Tag{Key: aws.String(k), Value: aws.String(v)})
	}
	return efsTags
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

func mergeSecurityGroups(securityGroups []*string) string {
	dereferenced := make([]string, 0, len(securityGroups))
	for _, d := range securityGroups {
		dereferenced = append(dereferenced, *d)
	}
	return strings.Join(dereferenced, securityGroupSeperator)
}

func mountTargetKey(mountTargetID string) string {
	return mountTargetKeyPrefix + mountTargetID
}

func mountTargetValue(subnetID string, securityGroups []*string) string {
	return subnetID + securityGroupSeperator + mergeSecurityGroups(securityGroups)
}
