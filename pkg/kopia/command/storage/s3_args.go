// Copyright 2022 The Kanister Authors.
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
	"strings"

	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/logsafe"
)

const (
	s3SubCommand           = "s3"
	s3DisableTLSFlag       = "--disable-tls"
	s3DisableTLSVerifyFlag = "--disable-tls-verification"
	s3EndpointFlag         = "--endpoint"
	s3RegionFlag           = "--region"
)

func s3Args(location map[string][]byte, repoPathPrefix string) logsafe.Cmd {
	args := logsafe.NewLoggable(s3SubCommand)
	args = args.AppendLoggableKV(bucketFlag, getBucketNameFromMap(location))

	e := getEndpointFromMap(location)
	if e != "" {
		s3Endpoint := ResolveS3Endpoint(e)
		args = args.AppendLoggableKV(s3EndpointFlag, s3Endpoint)

		if httpInsecureEndpoint(e) {
			args = args.AppendLoggable(s3DisableTLSFlag)
		}
	}

	// Append prefix from the location to the repository path prefix, if specified
	fullRepoPathPrefix := GenerateFullRepoPath(getPrefixFromMap(location), repoPathPrefix)

	args = args.AppendLoggableKV(prefixFlag, fullRepoPathPrefix)

	if checkSkipSSLVerifyFromMap(location) {
		args = args.AppendLoggable(s3DisableTLSVerifyFlag)
	}

	region := getRegionFromMap(location)
	if region != "" {
		args = args.AppendLoggableKV(s3RegionFlag, region)
	}

	return args
}

// ResolveS3Endpoint removes the trailing slash and
// protocol from provided endpoint and returns the absolute
// endpoint string
func ResolveS3Endpoint(endpoint string) string {
	if strings.HasSuffix(endpoint, "/") {
		log.Debug().Print("Removing trailing slashes from the endpoint")
		endpoint = strings.TrimRight(endpoint, "/")
	}
	sp := strings.SplitN(endpoint, "://", 2)
	if len(sp) > 1 {
		log.Debug().Print("Removing leading protocol from the endpoint")
	}
	return sp[len(sp)-1]
}

func httpInsecureEndpoint(endpoint string) bool {
	return strings.HasPrefix(endpoint, "http:")
}
