package s3

import (
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag"
)

//
// S3 flags.
//

// Bucket creates a new S3 bucket flag with a given bucket name.
func Bucket(bucket string) flag.Applier {
	return flag.NewStringFlag("--bucket", bucket)
}

// Endpoint creates a new S3 endpoint flag with a given endpoint.
func Endpoint(endpoint string) flag.Applier {
	return flag.NewStringFlag("--endpoint", endpoint)
}

// Prefix creates a new S3 prefix flag with a given prefix.
func Prefix(prefix string) flag.Applier {
	return flag.NewStringFlag("--prefix", prefix)
}

// Region creates a new S3 region flag with a given region.
func Region(region string) flag.Applier {
	return flag.NewStringFlag("--region", region)
}

// DisableTLS creates a new S3 disable TLS flag.
func DisableTLS(disable bool) flag.Applier {
	return flag.NewBoolFlag("--disable-tls", disable)
}

// DisableTLSVerify creates a new S3 disable TLS verification flag.
func DisableTLSVerify(disable bool) flag.Applier {
	return flag.NewBoolFlag("--disable-tls-verification", disable)
}
