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

package azure

import (
	"github.com/kanisterio/safecli"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/command"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/storage/model"
)

// New returns a builder for the Azure subcommand storage.
func New(f model.StorageFlag) (*safecli.Builder, error) {
	prefix := model.GenerateFullRepoPath(f.Location.Prefix(), f.RepoPathPrefix)
	return command.NewCommandBuilder(command.Azure,
		AzureCountainer(f.Location.BucketName()),
		Prefix(prefix),
	)
}
