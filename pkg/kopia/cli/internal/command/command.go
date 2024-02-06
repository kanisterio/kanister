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

package command

import (
	"github.com/kanisterio/safecli"

	clierrors "github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag"
)

// Command is a CLI command/subcommand.
type Command struct {
	name string
}

// Apply applies the command to the CLI.
func (c Command) Apply(cli safecli.CommandAppender) error {
	if len(c.name) == 0 {
		return clierrors.ErrInvalidCommand
	}
	cli.AppendLoggable(c.name)
	return nil
}

// NewCommandBuilder returns a new safecli.Builder for the storage sub command.
func NewCommandBuilder(cmd flag.Applier, flags ...flag.Applier) (*safecli.Builder, error) {
	b := safecli.NewBuilder()
	if err := flag.Apply(b, append([]flag.Applier{cmd}, flags...)...); err != nil {
		return nil, err
	}
	return b, nil
}
