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
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/logsafe"
)

const (
	gcsSubCommand       = "gcs"
	credentialsFileFlag = "--credentials-file"
)

func gcsArgs(location map[string][]byte, repoPathPrefix string) logsafe.Cmd {
	// Append prefix from the location to the repository path prefix, if specified
	fullRepoPathPrefix := GenerateFullRepoPath(getPrefixFromMap(location), repoPathPrefix)

	args := logsafe.NewLoggable(gcsSubCommand)
	args = args.AppendLoggableKV(bucketFlag, getBucketNameFromMap(location))
	args = args.AppendLoggableKV(credentialsFileFlag, consts.GoogleCloudCredsFilePath)
	return args.AppendLoggableKV(prefixFlag, fullRepoPathPrefix)
}
