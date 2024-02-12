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
)

// Applier applies flags/args to the command.
type Applier interface {
	// Apply applies the flags/args to the command.
	Apply(cli safecli.CommandAppender) error
}

// Apply appends multiple flags to the CLI.
// If any of the flags encounter an error during the Apply process,
// the error is returned and no changes are made to the CLI.
// If no error, the flags are appended to the CLI.
func Apply(cli safecli.CommandAppender, flags ...Applier) error {
	// create a new sub builder which will be used to apply the flags
	// to avoid mutating the CLI if an error is encountered.
	sub := safecli.NewBuilder()
	for _, flag := range flags {
		if flag == nil { // if the flag is nil, skip it
			continue
		}
		if err := flag.Apply(sub); err != nil {
			return err
		}
	}
	cli.Append(sub)
	return nil
}

// flags defines a collection of Flags.
type flags []Applier

// Apply applies the flags to the CLI.
func (flags flags) Apply(cli safecli.CommandAppender) error {
	return Apply(cli, flags...)
}

// NewFlags creates a new collection of flags.
func NewFlags(fs ...Applier) Applier {
	return flags(fs)
}

// simpleFlag is a simple implementation of the Applier interface.
type simpleFlag struct {
	err error
}

// Apply does nothing except return an error if one is set.
func (f simpleFlag) Apply(safecli.CommandAppender) error {
	return f.err
}

// EmptyFlag creates a new flag that does nothing.
// It is useful for creating a no-op flag when a condition is not met
// but Applier interface is required.
func EmptyFlag() Applier {
	return simpleFlag{}
}

// ErrorFlag creates a new flag that returns an error when applied.
// It is useful for creating a flag validation if a condition is not met.
func ErrorFlag(err error) Applier {
	return simpleFlag{err}
}
