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

// Applier is an interface for applying flags/args to a command.
type Applier interface {
	// Apply applies the flags/args to the command.
	Apply(cli safecli.CommandAppender) error
}

// Apply attaches multiple flags to the CLI.
// If any of the flags encounter an error during the Apply process,
// the error is returned and no changes are made to the CLI.
// If no error is encountered, the flags are appended to the CLI.
func Apply(cli safecli.CommandAppender, flags ...Applier) error {
	// create a new builder which will be used to apply the flags
	// to avoid mutating the CLI if an error is encountered.
	b := safecli.NewBuilder()
	for _, f := range flags {
		if f == nil {
			continue // if the flag is nil, skip it
		}
		if err := f.Apply(b); err != nil {
			return err
		}
	}
	cli.Append(b) // if no error, append the flags to the CLI
	return nil
}

// flags defines a collection of flags.
type flags []Applier

// Apply applies the flags to the CLI.
func (flags flags) Apply(cli safecli.CommandAppender) error {
	for _, flag := range flags {
		if err := flag.Apply(cli); err != nil {
			return err
		}
	}
	return nil
}

// NewFlags creates a new collection of flags.
func NewFlags(fs ...Applier) Applier {
	return flags(fs)
}

// errorFlag is a flag that does nothing.
type errorFlag struct {
	err error
}

func (f errorFlag) Apply(safecli.CommandAppender) error {
	return f.err
}

// DoNothingFlag creates a new void flag.
func DoNothingFlag() Applier {
	return errorFlag{}
}

// ErrorFlag creates a new flag that returns an error.
func ErrorFlag(err error) Applier {
	return errorFlag{err}
}
