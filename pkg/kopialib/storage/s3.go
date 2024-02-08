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

	kopialibutils "github.com/kanisterio/kanister/pkg/kopialib/utils"
	"github.com/kanisterio/kanister/pkg/utils"
	"github.com/kopia/kopia/repo/blob"
	"github.com/kopia/kopia/repo/blob/s3"
)

var _ Storage = &s3Storage{}

var requiredS3Arguments = []string{
	kopialibutils.BucketKey,
}

type s3Storage struct {
	options *s3.Options
	create  bool
}

func (s *s3Storage) New() (blob.Storage, error) {
	return s3.New(context.Background(), s.options, s.create)
}

func (s *s3Storage) WithCreate() {
	s.create = true
}

func (s *s3Storage) SetOptions(ctx context.Context, options map[string]string) error {
	err := validateCommonStorageArgs(options, TypeS3, requiredS3Arguments)
	if err != nil {
		return err
	}
	s.options = &s3.Options{
		BucketName:      options[kopialibutils.BucketKey],
		Endpoint:        options[kopialibutils.S3EndpointKey],
		Prefix:          options[kopialibutils.PrefixKey],
		Region:          options[kopialibutils.S3RegionKey],
		SessionToken:    options[kopialibutils.S3TokenKey],
		AccessKeyID:     options[kopialibutils.S3AccessKey],
		SecretAccessKey: options[kopialibutils.S3SecretAccessKey],
	}
	doNotUseTLS, err := utils.GetBoolOrDefault(options[kopialibutils.DoNotUseTLS], true)
	if err != nil {
		return err
	}
	s.options.DoNotUseTLS = doNotUseTLS
	doNotVerifyTLS, err := utils.GetBoolOrDefault(options[kopialibutils.DoNotVerifyTLS], true)
	if err != nil {
		return err
	}

	s.options.DoNotVerifyTLS = doNotVerifyTLS
	return nil
}
