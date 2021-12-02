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
		req.SetBackupVaultName(e.backupVaultName)
		req.SetRecoveryPointArn(id)

		desc, err := e.DescribeRecoveryPointWithContext(ctx, req)
		if isResourceNotFoundException(err) {
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
		req.SetBackupVaultName(e.backupVaultName)
		req.SetRecoveryPointArn(id)

		_, err := e.DescribeRecoveryPointWithContext(ctx, req)
		if isResourceNotFoundException(err) {
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

func (e *efs) waitUntilMountTargetIsDeleted(ctx context.Context, mountTargetID string) error {
	return poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		req := &awsefs.DescribeMountTargetsInput{}
		req.SetMountTargetId(mountTargetID)

		_, err := e.DescribeMountTargetsWithContext(ctx, req)
		if isMountTargetNotFound(err) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
		return false, nil
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
			return false, errors.Errorf("Restore job is not completed successfully (%s)\n", *resp.StatusMessage)
		default:
			return false, nil
		}
	})
}
