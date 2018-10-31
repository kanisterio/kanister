package objectstore

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pkg/errors"
)

const (
	bucketNotFound = "NotFound"
	noSuchBucket   = s3.ErrCodeNoSuchBucket
	gcsS3NotFound  = "not found"
)

func config(region string) *aws.Config {
	c := aws.NewConfig()
	if region != "" {
		return c.WithRegion(region)
	}
	return c
}

func isBucketNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	if awsErr, ok := errors.Cause(err).(awserr.Error); ok {
		code := awsErr.Code()
		return code == bucketNotFound || code == noSuchBucket
	}
	return strings.Contains(err.Error(), gcsS3NotFound)
}

func GetS3BucketRegion(ctx context.Context, bucketName, regionHint string) (string, error) {
	r := s3.NormalizeBucketLocation(regionHint)
	s, err := session.NewSession(config(r))
	if err != nil {
		return "", errors.Wrapf(err, "failed to create session, region = %s", r)
	}
	return s3manager.GetBucketRegion(ctx, s, bucketName, r)
}
