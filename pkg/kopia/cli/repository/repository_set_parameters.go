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
	flagrepo "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/repository"
)

// SetParametersArgs defines the arguments for the `kopia repository set-parameters ...` command.
type SetParametersArgs struct {
	cli.CommonArgs

	RetentionMode   string        // retention mode for supported storage backends
	RetentionPeriod time.Duration // retention period for supported storage backends

	Logger log.Logger
}

// SetParameters creates a new `kopia repository set-parameters ...` command.
func SetParameters(args SetParametersArgs) (safecli.CommandBuilder, error) {
	return command.NewKopiaCommandBuilder(args.CommonArgs,
		command.Repository, command.SetParameters,
		flagrepo.BlobRetention(args.RetentionMode, args.RetentionPeriod),
	)
}
