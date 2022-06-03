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

package awsefs

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/backup"
	awsefs "github.com/aws/aws-sdk-go/service/efs"
	"github.com/aws/aws-sdk-go/service/sts"
	uuid "github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/rand"

	awsconfig "github.com/kanisterio/kanister/pkg/aws"
	"github.com/kanisterio/kanister/pkg/blockstorage"
	kantags "github.com/kanisterio/kanister/pkg/blockstorage/tags"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
)

type Efs struct {
	*awsefs.EFS
	*backup.Backup
	accountID       string
	region          string
	role            string
	backupVaultName string
}

var _ blockstorage.Provider = (*Efs)(nil)

const (
	generalPurposePerformanceMode = awsefs.PerformanceModeGeneralPurpose
	defaultPerformanceMode        = generalPurposePerformanceMode

	burstingThroughputMode = awsefs.ThroughputModeBursting
	defaultThroughputMode  = burstingThroughputMode

	efsType    = "EFS"
	maxRetries = 10
)

var allowedMetadataKeys = map[string]bool{
	"file-system-id":  true,
	"Encrypted":       true,
	"KmsKeyId":        true,
	"PerformanceMode": true,
	"CreationToken":   true,
	"newFileSystem":   true,
}

// NewEFSProvider retuns a blockstorage provider for AWS EFS.
func NewEFSProvider(ctx context.Context, config map[string]string) (blockstorage.Provider, error) {
	awsConfig, region, err := awsconfig.GetConfig(ctx, config)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get configuration for EFS")
	}
	s, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create session for EFS")
	}
	stsCli := sts.New(s, aws.NewConfig().WithRegion(region).WithMaxRetries(maxRetries))
	user, err := stsCli.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get user")
	}
	if user.Account == nil {
		return nil, errors.New("Account ID is empty")
	}
	accountID := *user.Account
	efsCli := awsefs.New(s, aws.NewConfig().WithRegion(region).WithCredentials(awsConfig.Credentials).WithMaxRetries(maxRetries))
	backupCli := backup.New(s, aws.NewConfig().WithRegion(region).WithCredentials(awsConfig.Credentials).WithMaxRetries(maxRetries))

	efsVault, ok := config[awsconfig.ConfigEFSVaultName]
	if !ok || efsVault == "" {
		return nil, errors.New("EFS vault name is empty")
	}

	return &Efs{
		EFS:             efsCli,
		Backup:          backupCli,
		region:          region,
		accountID:       accountID,
		role:            config[awsconfig.ConfigRole],
		backupVaultName: efsVault,
	}, nil
}

func (e *Efs) Type() blockstorage.Type {
	return blockstorage.TypeEFS
}

// VolumeCreate implements interface method for EFS. It sends EFS volume create request
// to AWS EFS and waits until the file system is available. Eventually, it returns the
// volume info that is sent back from the AWS EFS.
func (e *Efs) VolumeCreate(ctx context.Context, volume blockstorage.Volume) (*blockstorage.Volume, error) {
	req := &awsefs.CreateFileSystemInput{}
	reqId, err := uuid.NewV4()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create UUID")
	}
	req.SetCreationToken(reqId.String())
	req.SetPerformanceMode(defaultPerformanceMode)
	req.SetThroughputMode(defaultThroughputMode)
	req.SetTags(convertToEFSTags(blockstorage.KeyValueToMap(volume.Tags)))

	fd, err := e.CreateFileSystemWithContext(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create EFS instance")
	}
	if fd.FileSystemId == nil {
		return nil, errors.New("Empty filesystem ID")
	}
	if err = e.waitUntilFileSystemAvailable(ctx, *fd.FileSystemId); err != nil {
		return nil, errors.Wrap(err, "EFS instance is not available")
	}
	vol, err := e.VolumeGet(ctx, *fd.FileSystemId, volume.Az)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get recently create EFS instance")
	}
	_, mountTargets, err := filterAndGetMountTargetsFromTags(blockstorage.KeyValueToMap(volume.Tags))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get filtered tags and mount targets")
	}
	if err = e.createMountTargets(ctx, vol.ID, mountTargets); err != nil {
		return nil, errors.Wrap(err, "Failed to create mount targets")
	}
	return vol, nil
}

func (e *Efs) VolumeCreateFromSnapshot(ctx context.Context, snapshot blockstorage.Snapshot, tags map[string]string) (*blockstorage.Volume, error) {
	reqM := &backup.GetRecoveryPointRestoreMetadataInput{}
	reqM.SetBackupVaultName(e.backupVaultName)
	reqM.SetRecoveryPointArn(snapshot.ID)

	respM, err := e.GetRecoveryPointRestoreMetadataWithContext(ctx, reqM)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get backup tag from recovery point directly")
	}
	rpTags := convertFromBackupTags(respM.RestoreMetadata)
	rp2Tags, err := e.getBackupTags(ctx, snapshot.ID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get backup tag from recovery point")
	}
	rpTags = kantags.Union(rpTags, rp2Tags)
	// RestorePoint tags has some tags to describe saved mount targets.
	// We need to get them and remove them from the tags
	filteredTags, mountTargets, err := filterAndGetMountTargetsFromTags(rpTags)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get filtered tags and mount targets")
	}
	// Add some metadata which are necessary for EFS restore to function properly.
	filteredTags = kantags.Union(filteredTags, efsRestoreTags())

	req := &backup.StartRestoreJobInput{}
	req.SetIamRoleArn(awsDefaultServiceBackupRole(e.accountID))

	// Start job only allows specific keys in metadata
	// https://docs.aws.amazon.com/aws-backup/latest/devguide/API_StartRestoreJob.html
	for k := range filteredTags {
		if !allowedMetadataKeys[k] {
			delete(filteredTags, k)
		}
	}
	req.SetMetadata(convertToBackupTags(filteredTags))
	req.SetRecoveryPointArn(snapshot.ID)
	req.SetResourceType(efsType)

	resp, err := e.StartRestoreJobWithContext(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to start the restore job")
	}
	if resp.RestoreJobId == nil {
		return nil, errors.New("Empty restore job ID")
	}
	restoreID := *resp.RestoreJobId
	if err = e.waitUntilRestoreComplete(ctx, restoreID); err != nil {
		return nil, errors.Wrap(err, "Restore job failed to complete")
	}
	respD := &backup.DescribeRestoreJobInput{}
	respD.SetRestoreJobId(restoreID)
	descJob, err := e.DescribeRestoreJobWithContext(ctx, respD)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get description for the restore job")
	}
	fsID, err := efsIDFromResourceARN(*descJob.CreatedResourceArn)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get filesystem ID")
	}
	if err = e.createMountTargets(ctx, fsID, mountTargets); err != nil {
		return nil, errors.Wrap(err, "Failed to create mount targets")
	}
	return e.VolumeGet(ctx, fsID, "")
}

func efsRestoreTags() map[string]string {
	return map[string]string{
		"newFileSystem":   "true",
		"CreationToken":   rand.String(16),
		"Encrypted":       "false",
		"PerformanceMode": generalPurposePerformanceMode,
	}
}

type mountTarget struct {
	subnetID       string
	securityGroups []string
}

type mountTargets map[string]*mountTarget

func (e *Efs) createMountTargets(ctx context.Context, fsID string, mts mountTargets) error {
	created := make([]*awsefs.MountTargetDescription, 0)
	for _, v := range mts {
		req := &awsefs.CreateMountTargetInput{}
		req.SetFileSystemId(fsID)
		req.SetSubnetId(v.subnetID)
		req.SetSecurityGroups(convertListOfStrings(v.securityGroups))

		mtd, err := e.CreateMountTargetWithContext(ctx, req)
		if err != nil {
			return errors.Wrap(err, "Failed to create mount target")
		}
		created = append(created, mtd)
	}

	for _, desc := range created {
		if err := e.waitUntilMountTargetReady(ctx, *desc.MountTargetId); err != nil {
			return errors.Wrap(err, "Failed while waiting for Mount target to be ready")
		}
	}
	return nil
}

func parseMountTargetKey(key string) (string, error) {
	if !strings.HasPrefix(key, mountTargetKeyPrefix) {
		return "", errors.New("Malformed string for mount target key")
	}
	return key[len(mountTargetKeyPrefix):], nil
}

func parseMountTargetValue(value string) (*mountTarget, error) {
	// Format:
	// String until the first "+" is subnetID
	// After that "+" separates security groups
	// Example value:
	// subnet-123+securityGroup-1+securityGroup-2
	tokens := strings.Split(value, securityGroupSeperator)
	if len(tokens) < 1 {
		return nil, errors.New("Malformed string for mount target values")
	}
	subnetID := tokens[0]
	sgs := make([]string, 0)
	if len(tokens) > 1 {
		sgs = append(sgs, tokens[1:]...)
	}
	return &mountTarget{
		subnetID:       subnetID,
		securityGroups: sgs,
	}, nil
}

func filterAndGetMountTargetsFromTags(tags map[string]string) (map[string]string, mountTargets, error) {
	filteredTags := make(map[string]string)
	mts := make(mountTargets)
	for k, v := range tags {
		if strings.HasPrefix(k, mountTargetKeyPrefix) {
			id, err := parseMountTargetKey(k)
			if err != nil {
				return nil, nil, err
			}
			mt, err := parseMountTargetValue(v)
			if err != nil {
				return nil, nil, err
			}
			mts[id] = mt
		} else {
			// It is not a mount target tag, so pass it
			filteredTags[k] = v
		}
	}
	return filteredTags, mts, nil
}

func (e *Efs) getBackupTags(ctx context.Context, arn string) (map[string]string, error) {
	result := make(map[string]string)
	for resp, req := emptyResponseRequestForListTags(); resp.NextToken != nil; req.NextToken = resp.NextToken {
		var err error
		req.SetResourceArn(arn)
		resp, err = e.ListTagsWithContext(ctx, req)
		if err != nil {
			return nil, err
		}
		tags := convertFromBackupTags(resp.Tags)
		result = kantags.Union(result, tags)
	}
	return result, nil
}

func (e *Efs) VolumeDelete(ctx context.Context, volume *blockstorage.Volume) error {
	mts, err := e.getMountTargets(ctx, volume.ID)
	if isVolumeNotFound(err) {
		return nil
	}
	err = e.deleteMountTargets(ctx, mts)
	if err != nil {
		return errors.Wrap(err, "Failed to delete mount targets")
	}

	req := &awsefs.DeleteFileSystemInput{}
	req.SetFileSystemId(volume.ID)
	output, err := e.DeleteFileSystemWithContext(ctx, req)
	if err == nil {
		log.Info().Print("Delete EFS output", field.M{"output": output.String()})
	}
	if isVolumeNotFound(err) {
		return nil
	}
	return err
}

func (e *Efs) getMountTargets(ctx context.Context, fsID string) ([]*awsefs.MountTargetDescription, error) {
	mts := make([]*awsefs.MountTargetDescription, 0)
	for resp, req := emptyResponseRequestForMountTargets(); resp.NextMarker != nil; req.Marker = resp.NextMarker {
		var err error
		req.SetFileSystemId(fsID)
		resp, err = e.DescribeMountTargetsWithContext(ctx, req)
		if err != nil {
			return nil, err
		}
		mts = append(mts, resp.MountTargets...)
	}
	return mts, nil
}

func (e *Efs) deleteMountTargets(ctx context.Context, mts []*awsefs.MountTargetDescription) error {
	for _, mt := range mts {
		req := &awsefs.DeleteMountTargetInput{}
		req.SetMountTargetId(*mt.MountTargetId)
		_, err := e.DeleteMountTargetWithContext(ctx, req)
		if err != nil {
			return err
		}
		err = e.waitUntilMountTargetIsDeleted(ctx, *mt.MountTargetId)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *Efs) VolumeGet(ctx context.Context, id string, zone string) (*blockstorage.Volume, error) {
	desc, err := e.getFileSystemDescriptionWithID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get EFS volume")
	}
	return volumeFromEFSDescription(desc, zone), nil
}

func (e *Efs) SnapshotCopy(ctx context.Context, from blockstorage.Snapshot, to blockstorage.Snapshot) (*blockstorage.Snapshot, error) {
	return nil, errors.New("Not implemented")
}

func (e *Efs) SnapshotCopyWithArgs(ctx context.Context, from blockstorage.Snapshot, to blockstorage.Snapshot, args map[string]string) (*blockstorage.Snapshot, error) {
	return nil, errors.New("Copy Snapshot with Args not implemented")
}

func (e *Efs) SnapshotCreate(ctx context.Context, volume blockstorage.Volume, tags map[string]string) (*blockstorage.Snapshot, error) {
	err := e.CreateBackupVaultWrapper()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to setup K10 vault for AWS Backup")
	}
	desc, err := e.getFileSystemDescriptionWithID(ctx, volume.ID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get corresponding description")
	}

	req := &backup.StartBackupJobInput{}
	req.SetBackupVaultName(e.backupVaultName)
	req.SetIamRoleArn(awsDefaultServiceBackupRole(e.accountID))
	req.SetResourceArn(resourceARNForEFS(e.region, *desc.OwnerId, *desc.FileSystemId))

	// Save mount points and security groups as tags
	infraTags, err := e.getMountPointAndSecurityGroupTags(ctx, volume.ID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get mount points and security groups")
	}
	allTags := kantags.Union(tags, infraTags)
	req.SetRecoveryPointTags(convertToBackupTags(allTags))
	resp, err := e.StartBackupJob(req)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to start a backup job")
	}
	if err = e.waitUntilRecoveryPointVisible(ctx, *resp.RecoveryPointArn); err != nil {
		return nil, errors.Wrap(err, "Failed to fetch recovery point")
	}
	if err = e.setBackupTags(ctx, *resp.RecoveryPointArn, infraTags); err != nil {
		return nil, errors.Wrap(err, "Failed to set backup tags")
	}

	req2 := &backup.DescribeRecoveryPointInput{}
	req2.SetBackupVaultName(e.backupVaultName)
	req2.SetRecoveryPointArn(*resp.RecoveryPointArn)
	describeRP, err := e.DescribeRecoveryPointWithContext(ctx, req2)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get recovery point information")
	}
	return &blockstorage.Snapshot{
		CreationTime: blockstorage.TimeStamp(*resp.CreationDate),
		Encrypted:    volume.Encrypted,
		ID:           *resp.RecoveryPointArn,
		Region:       e.region,
		SizeInBytes:  *describeRP.BackupSizeInBytes,
		Tags:         blockstorage.MapToKeyValue(allTags),
		Volume:       &volume,
		Type:         blockstorage.TypeEFS,
	}, nil
}

// Create a Backup Vault, also checks if vault already exist
func (e *Efs) CreateBackupVaultWrapper() error {
	req := &backup.CreateBackupVaultInput{}
	req.SetBackupVaultName(e.backupVaultName)

	_, err := e.CreateBackupVault(req)
	if isBackupVaultAlreadyExists(err) {
		return nil
	}
	return err
}

func (e *Efs) SnapshotCreateWaitForCompletion(ctx context.Context, snapshot *blockstorage.Snapshot) error {
	return e.waitUntilRecoveryPointCompleted(ctx, snapshot.ID)
}

func (e *Efs) SnapshotDelete(ctx context.Context, snapshot *blockstorage.Snapshot) error {
	req := &backup.DeleteRecoveryPointInput{}
	req.SetBackupVaultName(e.backupVaultName)
	req.SetRecoveryPointArn(snapshot.ID)

	output, err := e.DeleteRecoveryPointWithContext(ctx, req)
	if err == nil {
		log.Info().Print("Delete EFS snapshot", field.M{"output": output.String()})
	}
	if isResourceNotFoundException(err) {
		return nil
	}
	if isDeleteInProgress(err) {
		return nil
	}
	return err
}

func (e *Efs) SnapshotGet(ctx context.Context, id string) (*blockstorage.Snapshot, error) {
	req := &backup.DescribeRecoveryPointInput{}
	req.SetBackupVaultName(e.backupVaultName)
	req.SetRecoveryPointArn(id)

	resp, err := e.DescribeRecoveryPointWithContext(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get recovery point information")
	}
	if resp.ResourceArn == nil {
		return nil, errors.Wrap(err, "Resource ARN in recovery point is empty")
	}
	volID, err := efsIDFromResourceARN(*resp.ResourceArn)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get volume ID from recovery point ARN")
	}
	vol, err := e.VolumeGet(ctx, volID, "")
	if err != nil && !isVolumeNotFound(err) {
		return nil, errors.Wrap(err, "Failed to get filesystem")
	}
	return snapshotFromRecoveryPoint(resp, vol, e.region)
}

func (e *Efs) SetTags(ctx context.Context, resource interface{}, tags map[string]string) error {
	switch r := resource.(type) {
	case *blockstorage.Volume:
		return e.setEFSTags(ctx, r.ID, tags)
	case *blockstorage.Snapshot:
		return e.setBackupTags(ctx, r.ID, tags)
	default:
		return errors.New("Unsupported type for setting tags")
	}
}

func (e *Efs) setBackupTags(ctx context.Context, arn string, tags map[string]string) error {
	if len(tags) == 0 {
		return nil
	}
	req := &backup.TagResourceInput{
		ResourceArn: &arn,
		Tags:        convertToBackupTags(tags),
	}
	_, err := e.Backup.TagResourceWithContext(ctx, req)
	return err
}

func (e *Efs) setEFSTags(ctx context.Context, id string, tags map[string]string) error {
	if len(tags) == 0 {
		return nil
	}
	req := &awsefs.TagResourceInput{
		ResourceId: &id,
		Tags:       convertToEFSTags(tags),
	}
	_, err := e.EFS.TagResource(req)
	return err
}

func (e *Efs) VolumesList(ctx context.Context, tags map[string]string, zone string) ([]*blockstorage.Volume, error) {
	result := make([]*blockstorage.Volume, 0)
	for resp, req := emptyResponseRequestForFilesystems(); resp.NextMarker != nil; req.Marker = resp.NextMarker {
		var err error
		resp, err = e.DescribeFileSystemsWithContext(ctx, req)
		if err != nil {
			return nil, err
		}
		availables := filterAvailable(filterWithTags(resp.FileSystems, tags))
		result = append(result, volumesFromEFSDescriptions(availables, zone)...)
	}
	return result, nil
}

func (e *Efs) SnapshotsList(ctx context.Context, tags map[string]string) ([]*blockstorage.Snapshot, error) {
	result := make([]*blockstorage.Snapshot, 0)
	for resp, req := emptyResponseRequestForBackups(); resp.NextToken != nil; req.NextToken = resp.NextToken {
		var err error
		req.SetBackupVaultName(e.backupVaultName)
		resp, err = e.ListRecoveryPointsByBackupVaultWithContext(ctx, req)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to list recovery points by vault")
		}
		snaps, err := e.SnapshotsFromRecoveryPoints(ctx, resp.RecoveryPoints)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to get snapshots from recovery points")
		}
		result = append(result, blockstorage.FilterSnapshotsWithTags(snaps, tags)...)
	}
	return result, nil
}

// List a limited amount of snapshots based on given limit input
func (e *Efs) SnapshotsListWLimit(ctx context.Context, tags map[string]string, limit int64) ([]*blockstorage.Snapshot, error) {
	result := make([]*blockstorage.Snapshot, 0)
	var err error
	req := &backup.ListRecoveryPointsByBackupVaultInput{}
	req.SetBackupVaultName(e.backupVaultName)
	req.SetMaxResults(limit)
	resp, err := e.ListRecoveryPointsByBackupVaultWithContext(ctx, req) //backup API
	if err != nil {
		return nil, errors.Wrap(err, "Failed to list recovery points by vault")
	}
	snaps, err := e.SnapshotsFromRecoveryPoints(ctx, resp.RecoveryPoints)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get snapshots from recovery points")
	}
	result = append(result, blockstorage.FilterSnapshotsWithTags(snaps, tags)...)
	return result, err
}

func (e *Efs) SnapshotsFromRecoveryPoints(ctx context.Context, rps []*backup.RecoveryPointByBackupVault) ([]*blockstorage.Snapshot, error) {
	result := make([]*blockstorage.Snapshot, 0)
	for _, rp := range rps {
		if rp.RecoveryPointArn == nil {
			return nil, errors.New("Empty ARN in recovery point")
		}
		tags, err := e.getBackupTags(ctx, *rp.RecoveryPointArn)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to get backup tags")
		}
		volID, err := efsIDFromResourceARN(*rp.ResourceArn)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to get volume ID from recovery point ARN")
		}
		// VolumeGet might return error since originating filesystem might have
		// been deleted.
		vol, err := e.VolumeGet(ctx, volID, "")
		if err != nil && !isVolumeNotFound(err) {
			return nil, errors.Wrap(err, "Failed to get filesystem")
		}
		snap, err := snapshotFromRecoveryPointByVault(rp, vol, tags, e.region)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to get snapshot from the vault")
		}
		result = append(result, snap)
	}
	return result, nil
}

func emptyResponseRequestForBackups() (*backup.ListRecoveryPointsByBackupVaultOutput, *backup.ListRecoveryPointsByBackupVaultInput) {
	resp := (&backup.ListRecoveryPointsByBackupVaultOutput{}).SetNextToken("")
	req := &backup.ListRecoveryPointsByBackupVaultInput{}
	return resp, req
}

func emptyResponseRequestForFilesystems() (*awsefs.DescribeFileSystemsOutput, *awsefs.DescribeFileSystemsInput) {
	resp := (&awsefs.DescribeFileSystemsOutput{}).SetNextMarker("")
	req := &awsefs.DescribeFileSystemsInput{}
	return resp, req
}

func emptyResponseRequestForListTags() (*backup.ListTagsOutput, *backup.ListTagsInput) {
	resp := (&backup.ListTagsOutput{}).SetNextToken("")
	req := &backup.ListTagsInput{}
	return resp, req
}

func emptyResponseRequestForMountTargets() (*awsefs.DescribeMountTargetsOutput, *awsefs.DescribeMountTargetsInput) {
	resp := (&awsefs.DescribeMountTargetsOutput{}).SetNextMarker("")
	req := &awsefs.DescribeMountTargetsInput{}
	return resp, req
}

func awsDefaultServiceBackupRole(accountID string) string {
	return fmt.Sprintf("arn:aws:iam::%s:role/service-role/AWSBackupDefaultServiceRole", accountID)
}

func resourceARNForEFS(region string, accountID string, fileSystemID string) string {
	return fmt.Sprintf("arn:aws:elasticfilesystem:%s:%s:file-system/%s", region, accountID, fileSystemID)
}

func (e *Efs) getFileSystemDescriptionWithID(ctx context.Context, id string) (*awsefs.FileSystemDescription, error) {
	req := &awsefs.DescribeFileSystemsInput{}
	req.SetFileSystemId(id)

	descs, err := e.DescribeFileSystemsWithContext(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get filesystem description")
	}
	availables := filterAvailable(descs.FileSystems)
	switch len(availables) {
	case 0:
		return nil, errors.New("Failed to find volume")
	case 1:
		return descs.FileSystems[0], nil
	default:
		return nil, errors.New("Unexpected condition, multiple filesystems with same ID")
	}
}

func (e *Efs) getMountPointAndSecurityGroupTags(ctx context.Context, id string) (map[string]string, error) {
	mts, err := e.getMountTargets(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get mount target for the volume")
	}
	resultTags := make(map[string]string)
	for _, mt := range mts {
		req := &awsefs.DescribeMountTargetSecurityGroupsInput{}
		req.SetMountTargetId(*mt.MountTargetId)

		resp, err := e.DescribeMountTargetSecurityGroupsWithContext(ctx, req)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to get security group")
		}
		if mt.SubnetId == nil {
			return nil, errors.New("Empty subnet ID in mount target entry")
		}
		value := mountTargetValue(*mt.SubnetId, resp.SecurityGroups)
		if mt.MountTargetId == nil {
			return nil, errors.New("Empty ID in mount target entry")
		}
		key := mountTargetKey(*mt.MountTargetId)
		resultTags[key] = value
	}
	return resultTags, nil
}
