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
	"github.com/kopia/kopia/repo/blob/gcs"
)

type gcpStorage struct {
	Options *gcs.Options
	Create  bool
}

func (g *gcpStorage) Connect() (blob.Storage, error) {
	return gcs.New(context.Background(), g.Options, g.Create)
}

func (g *gcpStorage) WithOptions(opts gcs.Options) {
	g.Options = &opts
}

func (g *gcpStorage) WithCreate(create bool) {
	g.Create = create
}

func (g *gcpStorage) SetOptions(ctx context.Context, options map[string]string) {
	g.Options = &gcs.Options{
		Prefix:                        options[kopialib.PrefixKey],
		BucketName:                    options[kopialib.BucketKey],
		ServiceAccountCredentialsFile: options[kopialib.GCPServiceAccountCredentialsFile],
	}

	g.Options.ReadOnly, _ = utils.GetBoolOrDefault(options[kopialib.GCPReadOnly], false)
}
