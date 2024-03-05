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
	"github.com/kanisterio/safecli"

	"github.com/kanisterio/kanister/pkg/kopia/cli/args"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/opts"
	"github.com/kanisterio/kanister/pkg/log"
)

// StatusArgs defines the arguments for the `kopia repository status ...` command.
type StatusArgs struct {
	args.Common // embed common arguments

	JSONOutput bool // shows the output in JSON format

	Logger log.Logger
}

// Status creates a new `kopia repository status ...` command.
func Status(args StatusArgs) (*safecli.Builder, error) {
	return internal.NewKopiaCommand(
		opts.Common(args.Common),
		cmdRepository, subcmdStatus,
		opts.JSON(args.JSONOutput),
	)
}
