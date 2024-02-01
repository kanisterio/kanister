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
