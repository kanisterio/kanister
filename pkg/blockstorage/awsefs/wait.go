package awsefs

import (
	"context"

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
