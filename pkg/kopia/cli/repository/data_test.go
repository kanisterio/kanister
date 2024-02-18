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
