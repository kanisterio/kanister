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
	"github.com/go-openapi/strfmt"

	"github.com/kanisterio/safecli"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/command"
	flagcommon "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/common"
	flagrepo "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/repository"
	flagstorage "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/storage"
	"github.com/kanisterio/kanister/pkg/log"
)

// ConnectArgs defines the arguments for the `kopia repository connect` command.
type ConnectArgs struct {
	cli.CommonArgs
	cli.CacheArgs

	Hostname       string            // the hostname of the repository
	Username       string            // the username of the repository
	Location       map[string][]byte // the location of the repository
	RepoPathPrefix string            // the prefix of the repository path
	ReadOnly       bool              // connect to a repository in read-only mode
	PointInTime    strfmt.DateTime   // connect to a repository as it was at a specific point in time

	Logger log.Logger
}

// Connect creates a new `kopia repository connect ...` command.
func Connect(args ConnectArgs) (safecli.CommandBuilder, error) {
	return command.NewKopiaCommandBuilder(args.CommonArgs,
		command.Repository, command.Connect,
		flagcommon.NoCheckForUpdates,
		flagcommon.ReadOnly(args.ReadOnly),
		flagcommon.Cache(args.CacheArgs),
		flagrepo.Hostname(args.Hostname),
		flagrepo.Username(args.Username),
		flagstorage.Storage(args.Location, args.RepoPathPrefix, flagstorage.WithLogger(args.Logger)),
		flagrepo.PIT(args.PointInTime),
	)
}