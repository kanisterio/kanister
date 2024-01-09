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
	"crypto/tls"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pkg/errors"

	kaws "github.com/kanisterio/kanister/pkg/aws"
)

const (
	bucketNotFound = "NotFound"
	noSuchBucket   = s3.ErrCodeNoSuchBucket
	gcsS3NotFound  = "not found"
)

func IsBucketNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	if awsErr, ok := errors.Cause(err).(awserr.Error); ok {
		code := awsErr.Code()
		return code == bucketNotFound || code == noSuchBucket
	}
	return strings.Contains(err.Error(), gcsS3NotFound)
}

func awsConfig(ctx context.Context, pc ProviderConfig, s SecretAws) (*aws.Config, string, error) {
	c := map[string]string{
		kaws.AccessKeyID:     s.AccessKeyID,
		kaws.SecretAccessKey: s.SecretAccessKey,
		kaws.SessionToken:    s.SessionToken,
		kaws.ConfigRegion:    s3.NormalizeBucketLocation(pc.Region),
		//TODO: Add aws.ConfigRole to profile
	}
	cfg, r, err := kaws.GetConfig(ctx, c)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to create aws config")
	}
	cfg = cfg.WithRegion(r)
	if pc.Endpoint != "" {
		cfg = cfg.WithEndpoint(pc.Endpoint).WithS3ForcePathStyle(true)
	}
	if pc.SkipSSLVerify {
		h := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
		cfg = cfg.WithHTTPClient(h)
	}

	return cfg, r, nil
}
