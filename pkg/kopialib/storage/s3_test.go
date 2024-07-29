// Copyright 2024 The Kanister Authors.
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

package storage

import (
	"context"
	"fmt"

	kopiaerrors "github.com/kanisterio/kanister/pkg/kopia/errors"
	"github.com/kanisterio/kanister/pkg/kopialib/utils"
	"github.com/kopia/kopia/repo/blob/s3"
	. "gopkg.in/check.v1"
)

type s3StorageTestSuite struct{}

var _ = Suite(&s3StorageTestSuite{})

func (s *s3StorageTestSuite) TestSetOptions(c *C) {
	for i, tc := range []struct {
		name            string
		options         map[string]string
		expectedOptions s3.Options
		expectedErr     string
		errChecker      Checker
		desc            string
	}{
		{
			name:        "options not set",
			options:     map[string]string{},
			expectedErr: fmt.Sprintf(kopiaerrors.ErrStorageOptionsCannotBeNilMsg, TypeS3),
			errChecker:  NotNil,
			desc:        "options to connect to S3 storage not set",
		},
		{
			name: "bucket name is required",
			options: map[string]string{
				utils.S3EndpointKey: "test-endpoint",
			},
			expectedErr: fmt.Sprintf(kopiaerrors.ErrMissingRequiredFieldMsg, utils.BucketKey, TypeS3),
			errChecker:  NotNil,
			desc:        "required field `bucket name` is not set",
		},
		{
			name: "set correct options",
			options: map[string]string{
				utils.BucketKey:         "test-bucket",
				utils.S3EndpointKey:     "test-endpoint",
				utils.S3RegionKey:       "test-region",
				utils.S3AccessKey:       "test-access-key",
				utils.S3SecretAccessKey: "test-secret-access-key",
				utils.S3TokenKey:        "test-s3-token",
				utils.PrefixKey:         "test-prefix",
			},
			expectedOptions: s3.Options{
				BucketName:      "test-bucket",
				Endpoint:        "test-endpoint",
				Region:          "test-region",
				AccessKeyID:     "test-access-key",
				SecretAccessKey: "test-secret-access-key",
				Prefix:          "test-prefix",
				SessionToken:    "test-s3-token",
				DoNotUseTLS:     true,
				DoNotVerifyTLS:  true,
			},
			errChecker: IsNil,
			desc:       "All the connect options for S3 are set correctly",
		},
		{
			name: "set TLS",
			options: map[string]string{
				utils.BucketKey:         "test-bucket",
				utils.S3EndpointKey:     "test-endpoint",
				utils.S3RegionKey:       "test-region",
				utils.S3AccessKey:       "test-access-key",
				utils.S3SecretAccessKey: "test-secret-access-key",
				utils.S3TokenKey:        "test-s3-token",
				utils.PrefixKey:         "test-prefix",
				utils.DoNotUseTLS:       "false",
			},
			expectedOptions: s3.Options{
				BucketName:      "test-bucket",
				Endpoint:        "test-endpoint",
				Region:          "test-region",
				AccessKeyID:     "test-access-key",
				SecretAccessKey: "test-secret-access-key",
				Prefix:          "test-prefix",
				SessionToken:    "test-s3-token",
				DoNotUseTLS:     false,
				DoNotVerifyTLS:  true,
			},
			errChecker: IsNil,
			desc:       "Verify if TLS is set to true",
		},
		{
			name: "set verify TLS",
			options: map[string]string{
				utils.BucketKey:         "test-bucket",
				utils.S3EndpointKey:     "test-endpoint",
				utils.S3RegionKey:       "test-region",
				utils.S3AccessKey:       "test-access-key",
				utils.S3SecretAccessKey: "test-secret-access-key",
				utils.S3TokenKey:        "test-s3-token",
				utils.PrefixKey:         "test-prefix",
				utils.DoNotUseTLS:       "false",
				utils.DoNotVerifyTLS:    "false",
			},
			expectedOptions: s3.Options{
				BucketName:      "test-bucket",
				Endpoint:        "test-endpoint",
				Region:          "test-region",
				AccessKeyID:     "test-access-key",
				SecretAccessKey: "test-secret-access-key",
				Prefix:          "test-prefix",
				SessionToken:    "test-s3-token",
				DoNotUseTLS:     false,
				DoNotVerifyTLS:  false,
			},
			errChecker: IsNil,
			desc:       "Check if VerifyTLS is set to true",
		},
	} {
		s3Storage := s3Storage{}
		err := s3Storage.SetOptions(context.Background(), tc.options)
		c.Check(err, tc.errChecker)
		if err != nil {
			c.Check(err.Error(), Equals, tc.expectedErr, Commentf("test number: %d", i))
			c.Check(err.Error(), Equals, tc.expectedErr, Commentf("test number: %d, desc: %s", i, tc.desc))
		}

		c.Assert(s3Storage.options, DeepEquals, tc.expectedOptions)
	}
}
