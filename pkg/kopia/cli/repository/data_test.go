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

package repository

import (
	"time"

	"github.com/kanisterio/kanister/pkg/kopia/cli/args"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal"
	rs "github.com/kanisterio/kanister/pkg/secrets/repositoryserver"
)

var (
	common = args.Common{
		RepoPassword:   "encr-key",
		ConfigFilePath: "path/kopia.config",
		LogDirectory:   "cache/log",
	}

	cache = args.Cache{
		CacheDirectory:           "/tmp/cache.dir",
		ContentCacheSizeLimitMB:  0,
		MetadataCacheSizeLimitMB: 0,
	}
)

var (
	retentionMode   = "Locked"
	retentionPeriod = 15 * time.Minute

	locFS = internal.Location{
		rs.TypeKey:   []byte("filestore"),
		rs.PrefixKey: []byte("test-prefix"),
	}

	locAzure = internal.Location{
		rs.TypeKey:   []byte("azure"),
		rs.BucketKey: []byte("test-bucket"),
		rs.PrefixKey: []byte("test-prefix"),
	}

	locGCS = internal.Location{
		rs.TypeKey:   []byte("gcs"),
		rs.BucketKey: []byte("test-bucket"),
		rs.PrefixKey: []byte("test-prefix"),
	}

	locS3 = internal.Location{
		rs.TypeKey:          []byte("s3"),
		rs.EndpointKey:      []byte("test-endpoint"),
		rs.RegionKey:        []byte("test-region"),
		rs.BucketKey:        []byte("test-bucket"),
		rs.PrefixKey:        []byte("test-prefix"),
		rs.SkipSSLVerifyKey: []byte("false"),
	}

	locS3Compliant = internal.Location{
		rs.TypeKey:          []byte("s3Compliant"),
		rs.EndpointKey:      []byte("test-endpoint"),
		rs.RegionKey:        []byte("test-region"),
		rs.BucketKey:        []byte("test-bucket"),
		rs.PrefixKey:        []byte("test-prefix"),
		rs.SkipSSLVerifyKey: []byte("false"),
	}

	locFTP = internal.Location{
		rs.TypeKey: []byte("ftp"),
	}
)
