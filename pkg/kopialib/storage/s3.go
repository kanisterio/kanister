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

	"github.com/kanisterio/kanister/pkg/kopialib"
	"github.com/kanisterio/kanister/pkg/utils"
	"github.com/kopia/kopia/repo/blob"
	"github.com/kopia/kopia/repo/blob/s3"
)

var requiredS3Arguments = []string{
	kopialib.BucketKey,
	kopialib.S3EndpointKey,
	kopialib.S3AccessKey,
	kopialib.S3SecretAccessKey,
	kopialib.S3RegionKey,
}

type s3Storage struct {
	Options *s3.Options
	Create  bool
}

func (s *s3Storage) Connect() (blob.Storage, error) {
	return s3.New(context.Background(), s.Options, s.Create)
}

func (s *s3Storage) WithOptions(opts s3.Options) {
	s.Options = &opts
}

func (s *s3Storage) WithCreate(create bool) {
	s.Create = create
}

func (s *s3Storage) SetOptions(ctx context.Context, options map[string]string) error {
	err := validateCommonStorageArgs(options, TypeS3, requiredS3Arguments)
	if err != nil {
		return err
	}
	s.Options = &s3.Options{
		BucketName:      options[kopialib.BucketKey],
		Endpoint:        options[kopialib.S3EndpointKey],
		Prefix:          options[kopialib.PrefixKey],
		Region:          options[kopialib.S3RegionKey],
		SessionToken:    options[kopialib.S3TokenKey],
		AccessKeyID:     options[kopialib.S3AccessKey],
		SecretAccessKey: options[kopialib.S3SecretAccessKey],
	}
	s.Options.DoNotUseTLS, _ = utils.GetBoolOrDefault(options[kopialib.DoNotUseTLS], true)
	s.Options.DoNotVerifyTLS, _ = utils.GetBoolOrDefault(options[kopialib.DoNotVerifyTLS], true)

	return nil
}
