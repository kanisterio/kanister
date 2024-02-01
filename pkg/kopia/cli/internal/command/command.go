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

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag"

	flagcommon "github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag/common"
)

// Command is a CLI command/subcommand.
type Command string

// Apply applies the command to the CLI.
func (c Command) Apply(cli safecli.CommandAppender) error {
	cli.AppendLoggable(string(c))
	return nil
}

// KopiaBinaryName is the name of the Kopia binary.
const (
	KopiaBinaryName = Command("kopia")
)

// Repository commands.
const (
	Repository    = Command("repository")
	Create        = Command("create")
	Connect       = Command("connect")
	Server        = Command("server")
	Status        = Command("status")
	SetParameters = Command("set-parameters")
)

// Repository storage sub commands.
const (
	S3         = Command("s3")
	GCS        = Command("gcs")
	Azure      = Command("azure")
	FileSystem = Command("filesystem")
)

// NewKopiaCommandBuilder returns a new Kopia command builder.
func NewKopiaCommandBuilder(args cli.CommonArgs, flags ...flag.Applier) (*safecli.Builder, error) {
	flags = append([]flag.Applier{flagcommon.Common(args)}, flags...)
	return NewCommandBuilder(KopiaBinaryName, flags...)
}

// NewCommandBuilder returns a new safecli.Builder for the storage sub command.
func NewCommandBuilder(cmd flag.Applier, flags ...flag.Applier) (*safecli.Builder, error) {
	b := safecli.NewBuilder()
	if err := cmd.Apply(b); err != nil {
		return nil, err
	}
	if err := flag.Apply(b, flags...); err != nil {
		return nil, err
	}
	return b, nil
}
