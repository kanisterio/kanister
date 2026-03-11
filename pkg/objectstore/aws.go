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

package objectstore

import (
	"context"
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/smithy-go"
	"github.com/kanisterio/errkit"

	kaws "github.com/kanisterio/kanister/pkg/aws"
)

const (
	bucketNotFound = "NotFound"
	noSuchBucket   = "NoSuchBucket"
	gcsS3NotFound  = "not found"
)

func IsBucketNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	// Check for AWS SDK v2 error types
	var noSuchBucketErr *s3types.NoSuchBucket
	if errors.As(err, &noSuchBucketErr) {
		return true
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.ErrorCode()
		return code == bucketNotFound || code == noSuchBucket
	}
	// Check for AWS SDK v1 error types (stow still uses v1)
	var awsErr awserr.Error
	if errors.As(err, &awsErr) {
		code := awsErr.Code()
		return code == bucketNotFound || code == noSuchBucket
	}
	return strings.Contains(err.Error(), gcsS3NotFound)
}

// normalizeBucketLocation maps an empty location string to "us-east-1",
// mirroring the v1 SDK's s3.NormalizeBucketLocation behaviour.
func normalizeBucketLocation(region string) string {
	if region == "" {
		return "us-east-1"
	}
	return region
}

func awsConfig(ctx context.Context, pc ProviderConfig, s SecretAws) (aws.Config, string, error) {
	c := map[string]string{
		kaws.AccessKeyID:     s.AccessKeyID,
		kaws.SecretAccessKey: s.SecretAccessKey,
		kaws.SessionToken:    s.SessionToken,
		kaws.ConfigRegion:    normalizeBucketLocation(pc.Region),
		//TODO: Add aws.ConfigRole to profile
	}
	cfg, r, err := kaws.GetConfig(ctx, c)
	if err != nil {
		return aws.Config{}, "", errkit.Wrap(err, "failed to create aws config")
	}
	cfg.Region = r
	// Endpoint and SkipSSLVerify are applied at S3 client construction time (see bucket.go),
	// not on aws.Config in v2.
	return cfg, r, nil
}
