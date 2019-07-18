package awsefs

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	awsarn "github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/backup"
	awsefs "github.com/aws/aws-sdk-go/service/efs"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/pkg/errors"

	"github.com/kanisterio/kanister/pkg/blockstorage"
	"github.com/kanisterio/kanister/pkg/blockstorage/awsebs"
)

type efs struct {
	*awsefs.EFS
	*backup.Backup
	accountID string
	region    string
}

var _ blockstorage.Provider = (*efs)(nil)

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

func (e *efs) VolumeCreate(context.Context, blockstorage.Volume) (*blockstorage.Volume, error) {
	return nil, errors.New("Not implemented")
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

func (e *efs) getFileSystemDescriptionWithID(ctx context.Context, id string) (*awsefs.FileSystemDescription, error) {
	req := &awsefs.DescribeFileSystemsInput{}
	req.SetFileSystemId(id)

	descs, err := e.DescribeFileSystemsWithContext(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get filesystem description")
	}
	availables := filterAvailable(descs.FileSystems)
	var desc *awsefs.FileSystemDescription
	switch len(availables) {
	case 0:
		return nil, errors.New("Failed to find volume")
	case 1:
		desc = descs.FileSystems[0]
	default:
		return nil, errors.New("Unexpected condition, multiple filesystems with same ID")
	}
	return desc, nil
}
