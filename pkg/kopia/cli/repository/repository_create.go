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

	"github.com/kanisterio/safecli"

	"github.com/kanisterio/kanister/pkg/log"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/command"
	flagcommon "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/common"
	flagrepo "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/repository"
	flagstorage "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/storage"
)

// CreateArgs defines the arguments for the `kopia repository create` command.
type CreateArgs struct {
	cli.CommonArgs
	cli.CacheArgs

	Hostname        string            // the hostname of the repository
	Username        string            // the username of the repository
	Location        map[string][]byte // the location of the repository
	RepoPathPrefix  string            // the prefix of the repository path
	RetentionMode   string            // retention mode for supported storage backends
	RetentionPeriod time.Duration     // retention period for supported storage backends

	Logger log.Logger
}

// Create creates a new `kopia repository create ...` command.
func Create(args CreateArgs) (safecli.CommandBuilder, error) {
	return command.NewKopiaCommandBuilder(args.CommonArgs,
		command.Repository, command.Create,
		flagcommon.NoCheckForUpdates,
		flagcommon.Cache(args.CacheArgs),
		flagrepo.Hostname(args.Hostname),
		flagrepo.Username(args.Username),
		flagrepo.BlobRetention(args.RetentionMode, args.RetentionPeriod),
		flagstorage.Storage(
			args.Location,
			args.RepoPathPrefix,
			flagstorage.WithLogger(args.Logger), // log.Debug for old output
		),
	)
}
