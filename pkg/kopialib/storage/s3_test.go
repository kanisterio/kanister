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

	"github.com/kanisterio/kanister/pkg/kopialib"
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
	}{
		{
			name:        "options not set",
			options:     map[string]string{},
			expectedErr: fmt.Sprintf(ErrStorageOptionsCannotBeNilMsg, TypeS3),
			errChecker:  NotNil,
		},
		{
			name: "bucket name is required",
			options: map[string]string{
				kopialib.S3EndpointKey: "test-endpoint",
			},
			expectedErr: fmt.Sprintf(ErrMissingRequiredFieldMsg, kopialib.BucketKey, TypeS3),
			errChecker:  NotNil,
		},
		{
			name: "set correct options",
			options: map[string]string{
				kopialib.BucketKey:         "test-bucket",
				kopialib.S3EndpointKey:     "test-endpoint",
				kopialib.S3RegionKey:       "test-region",
				kopialib.S3AccessKey:       "test-access-key",
				kopialib.S3SecretAccessKey: "test-secret-access-key",
				kopialib.S3TokenKey:        "test-s3-token",
				kopialib.PrefixKey:         "test-prefix",
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
		},
		{
			name: "set TLS",
			options: map[string]string{
				kopialib.BucketKey:         "test-bucket",
				kopialib.S3EndpointKey:     "test-endpoint",
				kopialib.S3RegionKey:       "test-region",
				kopialib.S3AccessKey:       "test-access-key",
				kopialib.S3SecretAccessKey: "test-secret-access-key",
				kopialib.S3TokenKey:        "test-s3-token",
				kopialib.PrefixKey:         "test-prefix",
				kopialib.DoNotUseTLS:       "false",
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
		},
		{
			name: "set verify TLS",
			options: map[string]string{
				kopialib.BucketKey:         "test-bucket",
				kopialib.S3EndpointKey:     "test-endpoint",
				kopialib.S3RegionKey:       "test-region",
				kopialib.S3AccessKey:       "test-access-key",
				kopialib.S3SecretAccessKey: "test-secret-access-key",
				kopialib.S3TokenKey:        "test-s3-token",
				kopialib.PrefixKey:         "test-prefix",
				kopialib.DoNotUseTLS:       "false",
				kopialib.DoNotVerifyTLS:    "false",
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
		},
	} {
		s3Storage := s3Storage{}
		err := s3Storage.SetOptions(context.Background(), tc.options)
		c.Check(err, tc.errChecker)
		if err != nil {
			c.Check(err.Error(), Equals, tc.expectedErr, Commentf("test number: %d", i))
		}

		c.Assert(s3Storage.Options, DeepEquals, tc.expectedOptions)
	}
}
