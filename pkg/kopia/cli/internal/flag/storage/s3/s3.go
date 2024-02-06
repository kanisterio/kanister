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
	"strings"

	"github.com/kanisterio/safecli"

	"github.com/kanisterio/kanister/pkg/log"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/command"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/storage/model"
)

// New returns a builder for the S3 subcommand storage.
func New(s model.StorageFlag) (*safecli.Builder, error) {
	endpoint := resolveS3Endpoint(s.Location.Endpoint(), s.GetLogger())
	prefix := model.GenerateFullRepoPath(s.Location.Prefix(), s.RepoPathPrefix)
	return command.NewCommandBuilder(command.S3,
		Region(s.Location.Region()),
		Bucket(s.Location.BucketName()),
		Endpoint(endpoint),
		Prefix(prefix),
		DisableTLS(s.Location.IsInsecureEndpoint()),
		DisableTLSVerify(s.Location.HasSkipSSLVerify()),
	)
}

// resolveS3Endpoint removes the trailing slash and
// protocol from provided endpoint and
// returns the absolute endpoint string.
func resolveS3Endpoint(endpoint string, logger log.Logger) string {
	if endpoint == "" {
		return ""
	}

	if strings.HasSuffix(endpoint, "/") {
		logger.Print("Removing trailing slashes from the endpoint")
		endpoint = strings.TrimRight(endpoint, "/")
	}

	sp := strings.SplitN(endpoint, "://", 2)
	if len(sp) > 1 {
		logger.Print("Removing leading protocol from the endpoint")
	}

	return sp[len(sp)-1]
}
