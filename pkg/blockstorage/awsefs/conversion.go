// Copyright 2019 Kasten Inc.
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
	if rp == nil {
		return nil, errors.New("Empty recovery point")
	}
	if rp.CreationDate == nil {
		return nil, errors.New("Recovery point has no CreationDate")
	}
	if rp.BackupSizeInBytes == nil {
		return nil, errors.New("Recovery point has no BackupSizeInBytes")
	}
	if rp.RecoveryPointArn == nil {
		return nil, errors.New("Recovery point has no RecoveryPointArn")
	}
	encrypted := false
	if volume != nil {
		encrypted = volume.Encrypted
	}
	return &blockstorage.Snapshot{
		ID:           *rp.RecoveryPointArn,
		CreationTime: blockstorage.TimeStamp(*rp.CreationDate),
		Size:         bytesInGiB(*rp.BackupSizeInBytes),
		Region:       region,
		Type:         blockstorage.TypeEFS,
		Volume:       volume,
		Encrypted:    encrypted,
		Tags:         nil,
	}, nil
}

func snapshotFromRecoveryPointByVault(rp *backup.RecoveryPointByBackupVault, volume *blockstorage.Volume, tags map[string]string, region string) (*blockstorage.Snapshot, error) {
	if rp == nil {
		return nil, errors.New("Empty recovery point")
	}
	if rp.CreationDate == nil {
		return nil, errors.New("Recovery point has not CreationDate")
	}
	if rp.BackupSizeInBytes == nil {
		return nil, errors.New("Recovery point has no BackupSizeInBytes")
	}
	if rp.RecoveryPointArn == nil {
		return nil, errors.New("Recovery point has no RecoveryPointArn")
	}
	encrypted := false
	if volume != nil {
		encrypted = volume.Encrypted
	}
	return &blockstorage.Snapshot{
		ID:           *rp.RecoveryPointArn,
		CreationTime: blockstorage.TimeStamp(*rp.CreationDate),
		Size:         bytesInGiB(*rp.BackupSizeInBytes),
		Region:       region,
		Type:         blockstorage.TypeEFS,
		Volume:       volume,
		Encrypted:    encrypted,
		Tags:         blockstorage.MapToKeyValue(tags),
	}, nil
}

// convertFromBackupTags converts an AWS Backup compliant tag structure to a flattenned map.
func convertFromBackupTags(tags map[string]*string) map[string]string {
	result := make(map[string]string)
	for k, v := range tags {
		result[k] = *v
	}
	return result
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

// convertListOfStrings converts a flattend list to a list where each
// element is a pointer to original elements.
func convertListOfStrings(strs []string) []*string {
	result := make([]*string, 0)
	for i := range strs {
		result = append(result, &strs[i])
	}
	return result
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
