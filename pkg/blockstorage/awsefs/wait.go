package awsefs

import (
	"context"

	"github.com/aws/aws-sdk-go/service/backup"
	awsefs "github.com/aws/aws-sdk-go/service/efs"
	"github.com/pkg/errors"

	"github.com/kanisterio/kanister/pkg/poll"
)

const (
	maxNumErrorRetries = 3
)

func (e *efs) waitUntilFileSystemAvailable(ctx context.Context, id string) error {
	return poll.WaitWithRetries(ctx, maxNumErrorRetries, poll.IsAlwaysRetryable, func(ctx context.Context) (bool, error) {
		req := &awsefs.DescribeFileSystemsInput{}
		req.SetFileSystemId(id)

		desc, err := e.DescribeFileSystemsWithContext(ctx, req)
		if err != nil {
			return false, err
		}
		if len(desc.FileSystems) == 0 {
			return false, nil
		}
		state := desc.FileSystems[0].LifeCycleState
		if state == nil {
			return false, nil
		}
		return *state == awsefs.LifeCycleStateAvailable, nil
	})
}

func (e *efs) waitUntilRecoveryPointCompleted(ctx context.Context, id string) error {
	return poll.WaitWithRetries(ctx, maxNumErrorRetries, poll.IsAlwaysRetryable, func(ctx context.Context) (bool, error) {
		req := &backup.DescribeRecoveryPointInput{}
		req.SetBackupVaultName(k10BackupVaultName)
		req.SetRecoveryPointArn(id)

		desc, err := e.DescribeRecoveryPointWithContext(ctx, req)
		if isRecoveryPointNotFound(err) {
			// Recovery point doesn't appear when the backup jobs finishes.
			// Since this case is special, it will be counted as non-error.
			return false, nil
		}
		if err != nil {
			return false, err
		}
		status := desc.Status
		if status == nil {
			return false, nil
		}
		return *status == backup.RecoveryPointStatusCompleted, nil
	})
}

func (e *efs) waitUntilRecoveryPointVisible(ctx context.Context, id string) error {
	return poll.WaitWithRetries(ctx, maxNumErrorRetries, poll.IsAlwaysRetryable, func(ctx context.Context) (bool, error) {
		req := &backup.DescribeRecoveryPointInput{}
		req.SetBackupVaultName(k10BackupVaultName)
		req.SetRecoveryPointArn(id)

		_, err := e.DescribeRecoveryPointWithContext(ctx, req)
		if isRecoveryPointNotFound(err) {
			// Recovery point doesn't appear when the backup jobs finishes.
			// Since this case is special, it will be counted as non-error.
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return true, nil
	})
}

func (e *efs) waitUntilMountTargetReady(ctx context.Context, mountTargetID string) error {
	return poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		req := &awsefs.DescribeMountTargetsInput{}
		req.SetMountTargetId(mountTargetID)

		desc, err := e.DescribeMountTargetsWithContext(ctx, req)
		if isMountTargetNotFound(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if len(desc.MountTargets) != 1 {
			return false, errors.New("Returned list must have 1 entry")
		}
		mt := desc.MountTargets[0]
		state := mt.LifeCycleState
		if state == nil {
			return false, nil
		}
		return *state == awsefs.LifeCycleStateAvailable, nil
	})
}

func (e *efs) waitUntilRestoreComplete(ctx context.Context, restoreJobID string) error {
	return poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		req := &backup.DescribeRestoreJobInput{}
		req.SetRestoreJobId(restoreJobID)

		resp, err := e.DescribeRestoreJobWithContext(ctx, req)
		if err != nil {
			return false, err
		}
		if resp.Status == nil {
			return false, errors.New("Failed to get restore job status")
		}
		switch *resp.Status {
		case backup.RestoreJobStatusCompleted:
			return true, nil
		case backup.RestoreJobStatusAborted, backup.RestoreJobStatusFailed:
			return false, errors.New("Restore job is not completed successfully")
		default:
			return false, nil
		}
	})
}
