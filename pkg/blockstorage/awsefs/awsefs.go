package awsefs

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	awsarn "github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/backup"
	awsefs "github.com/aws/aws-sdk-go/service/efs"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"

	"github.com/kanisterio/kanister/pkg/blockstorage"
	"github.com/kanisterio/kanister/pkg/blockstorage/awsebs"
	kantags "github.com/kanisterio/kanister/pkg/blockstorage/tags"
)

type efs struct {
	*awsefs.EFS
	*backup.Backup
	accountID string
	region    string
}

var _ blockstorage.Provider = (*efs)(nil)

const (
	generalPurposePerformanceMode = awsefs.PerformanceModeGeneralPurpose
	maximumIOPerformanceMode      = awsefs.PerformanceModeMaxIo
	defaultPerformanceMode        = generalPurposePerformanceMode

	burstingThroughputMode    = awsefs.ThroughputModeBursting
	provisionedThroughputMode = awsefs.ThroughputModeProvisioned
	defaultThroughputMode     = burstingThroughputMode

	k10BackupVaultName = "k10vault"
	dummyMarker        = ""
)

// NewEFSProvider retuns a blockstorage provider for AWS EFS.
func NewEFSProvider(config map[string]string) (blockstorage.Provider, error) {
	awsConfig, region, err := awsebs.GetConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get configuration for EFS")
	}
	s, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create session for EFS")
	}
	iamCli := iam.New(s, aws.NewConfig().WithRegion(region))
	user, err := iamCli.GetUser(&iam.GetUserInput{})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get user")
	}
	if user.User == nil {
		return nil, errors.New("Failed to infer user from credentials")
	}
	userARN, err := awsarn.Parse(*user.User.Arn)
	if err != nil {
		return nil, err
	}
	accountID := userARN.AccountID
	efsCli := awsefs.New(s, aws.NewConfig().WithRegion(region))
	backupCli := backup.New(s, aws.NewConfig().WithRegion(region))
	return &efs{
		EFS:       efsCli,
		Backup:    backupCli,
		region:    region,
		accountID: accountID}, nil
}

func (e *efs) Type() blockstorage.Type {
	return blockstorage.TypeEFS
}

// VolumeCreate implements interface method for EFS. It sends EFS volume create request
// to AWS EFS and waits until the file system is available. Eventually, it returns the
// volume info that is sent back from the AWS EFS.
func (e *efs) VolumeCreate(ctx context.Context, volume blockstorage.Volume) (*blockstorage.Volume, error) {
	req := &awsefs.CreateFileSystemInput{}
	req.SetCreationToken(uuid.NewV4().String())
	req.SetPerformanceMode(defaultPerformanceMode)
	req.SetThroughputMode(defaultThroughputMode)
	req.SetTags(convertToEFSTags(blockstorage.KeyValueToMap(volume.Tags)))

	fd, err := e.CreateFileSystemWithContext(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create EFS instance")
	}
	if err = e.waitUntilFileSystemAvailable(ctx, *fd.FileSystemId); err != nil {
		return nil, errors.Wrap(err, "EFS instance is not available")
	}
	vol, err := e.VolumeGet(ctx, volume.ID, volume.Az)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get recently create EFS instance")
	}
	return vol, nil
}

func (e *efs) VolumeCreateFromSnapshot(ctx context.Context, snapshot blockstorage.Snapshot, tags map[string]string) (*blockstorage.Volume, error) {
	return nil, errors.New("Not implemented")
}

func (e *efs) VolumeDelete(ctx context.Context, volume *blockstorage.Volume) error {
	req := &awsefs.DeleteFileSystemInput{}
	req.SetFileSystemId(volume.ID)

	_, err := e.DeleteFileSystemWithContext(ctx, req)
	if isVolumeNotFound(err) {
		return nil
	}
	return err
}

func (e *efs) VolumeGet(ctx context.Context, id string, zone string) (*blockstorage.Volume, error) {
	desc, err := e.getFileSystemDescriptionWithID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get EFS volume")
	}
	return volumeFromEFSDescription(desc, zone), nil
}

func (e *efs) SnapshotCopy(ctx context.Context, from blockstorage.Snapshot, to blockstorage.Snapshot) (*blockstorage.Snapshot, error) {
	return nil, errors.New("Not implemented")
}

func (e *efs) SnapshotCreate(ctx context.Context, volume blockstorage.Volume, tags map[string]string) (*blockstorage.Snapshot, error) {
	err := e.createK10DefaultBackupVault()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to setup K10 vault for AWS Backup")
	}
	desc, err := e.getFileSystemDescriptionWithID(ctx, volume.ID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get corresponding description")
	}

	req := &backup.StartBackupJobInput{}
	req.SetBackupVaultName(k10BackupVaultName)
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
	return &blockstorage.Snapshot{
		CreationTime: blockstorage.TimeStamp(*resp.CreationDate),
		Encrypted:    volume.Encrypted,
		ID:           *resp.RecoveryPointArn,
		Region:       e.region,
		Size:         volume.Size,
		Tags:         blockstorage.MapToKeyValue(allTags),
		Volume:       &volume,
		Type:         blockstorage.TypeEFS,
	}, nil
}

func (e *efs) createK10DefaultBackupVault() error {
	req := &backup.CreateBackupVaultInput{}
	req.SetBackupVaultName(k10BackupVaultName)

	_, err := e.CreateBackupVault(req)
	if isBackupVaultAlreadyExists(err) {
		return nil
	}
	return err
}

func (e *efs) SnapshotCreateWaitForCompletion(ctx context.Context, snapshot *blockstorage.Snapshot) error {
	return e.waitUntilRecoveryPointCompleted(ctx, snapshot.ID)
}

func (e *efs) SnapshotDelete(ctx context.Context, snapshot *blockstorage.Snapshot) error {
	req := &backup.DeleteRecoveryPointInput{}
	req.SetBackupVaultName(k10BackupVaultName)
	req.SetRecoveryPointArn(snapshot.ID)

	_, err := e.DeleteRecoveryPointWithContext(ctx, req)
	return err
}

func (e *efs) SnapshotGet(ctx context.Context, id string) (*blockstorage.Snapshot, error) {
	req := &backup.DescribeRecoveryPointInput{}
	req.SetBackupVaultName(k10BackupVaultName)
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
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get originating volume")
	}
	return snapshotFromRecoveryPoint(resp, vol, e.region)
}

func (e *efs) SetTags(ctx context.Context, resource interface{}, tags map[string]string) error {
	switch r := resource.(type) {
	case *blockstorage.Volume:
		return e.setEFSTags(ctx, r.ID, tags)
	case *blockstorage.Snapshot:
		return e.setBackupTags(ctx, r.ID, tags)
	default:
		return errors.New("Unsupported type for setting tags")
	}
}

func (e *efs) setBackupTags(ctx context.Context, arn string, tags map[string]string) error {
	req := &backup.TagResourceInput{
		ResourceArn: &arn,
		Tags:        convertToBackupTags(tags),
	}
	_, err := e.TagResourceWithContext(ctx, req)
	return err
}

func (e *efs) setEFSTags(ctx context.Context, id string, tags map[string]string) error {
	req := &awsefs.CreateTagsInput{
		FileSystemId: &id,
		Tags:         convertToEFSTags(tags),
	}
	_, err := e.CreateTagsWithContext(ctx, req)
	return err
}

func (e *efs) VolumesList(ctx context.Context, tags map[string]string, zone string) ([]*blockstorage.Volume, error) {
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

func (e *efs) SnapshotsList(ctx context.Context, tags map[string]string) ([]*blockstorage.Snapshot, error) {
	return nil, errors.New("Not implemented")
}

func emptyResponseRequestForFilesystems() (*awsefs.DescribeFileSystemsOutput, *awsefs.DescribeFileSystemsInput) {
	resp := (&awsefs.DescribeFileSystemsOutput{}).SetNextMarker(dummyMarker)
	req := &awsefs.DescribeFileSystemsInput{}
	return resp, req
}

func emptyResponseRequestForMountTargets() (*awsefs.DescribeMountTargetsOutput, *awsefs.DescribeMountTargetsInput) {
	resp := (&awsefs.DescribeMountTargetsOutput{}).SetNextMarker(dummyMarker)
	req := &awsefs.DescribeMountTargetsInput{}
	return resp, req
}

func awsDefaultServiceBackupRole(accountID string) string {
	return fmt.Sprintf("arn:aws:iam::%s:role/service-role/AWSBackupDefaultServiceRole", accountID)
}

func resourceARNForEFS(region string, accountID string, fileSystemID string) string {
	return fmt.Sprintf("arn:aws:elasticfilesystem:%s:%s:file-system/%s", region, accountID, fileSystemID)
}

func (e *efs) getFileSystemDescriptionWithID(ctx context.Context, id string) (*awsefs.FileSystemDescription, error) {
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

func (e *efs) getMountPointAndSecurityGroupTags(ctx context.Context, id string) (map[string]string, error) {
	mts := make([]*awsefs.MountTargetDescription, 0)
	for resp, req := emptyResponseRequestForMountTargets(); resp.NextMarker != nil; req.Marker = resp.NextMarker {
		var err error
		req.SetFileSystemId(id)
		resp, err = e.DescribeMountTargetsWithContext(ctx, req)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to get mount targets")
		}
		mts = append(mts, resp.MountTargets...)
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
