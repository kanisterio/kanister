package awsefs

import (
	"context"

	"github.com/aws/aws-sdk-go/service/backup"
	awsefs "github.com/aws/aws-sdk-go/service/efs"

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
