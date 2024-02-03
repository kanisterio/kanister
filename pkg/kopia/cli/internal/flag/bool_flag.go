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

package flag

import (
	"github.com/kanisterio/safecli"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
)

// boolFlag defines a boolean flag with a given flag name.
// If enabled is set to true, the flag is applied; otherwise, it is not.
type boolFlag struct {
	flag    string
	enabled bool
}

// Apply appends the flag to the command if the flag is enabled.
func (f boolFlag) Apply(cli safecli.CommandAppender) error {
	if f.enabled {
		cli.AppendLoggable(f.flag)
	}
	return nil
}

// NewBoolFlag creates a new bool flag with a given flag name.
// If the flag name is empty, cli.ErrInvalidFlag is returned.
func NewBoolFlag(flag string, enabled bool) Applier {
	if flag == "" {
		return ErrorFlag(cli.ErrInvalidFlag)
	}
	return boolFlag{flag, enabled}
}
