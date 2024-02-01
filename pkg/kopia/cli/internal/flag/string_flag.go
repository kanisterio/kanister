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

// stringFlag defines a string flag with a given flag name and value.
// If the value is empty, the flag is not applied.
type stringFlag struct {
	flag     string // flag name
	value    string // flag value
	redacted bool   // output the value as redacted
}

// appenderFunc is a function that appends strings to a command.
type appenderFunc func(...string) *safecli.Builder

// Apply appends the flag to the command if the value is not empty.
// If the value is redacted, it is appended as redacted.
func (f stringFlag) Apply(cli safecli.CommandAppender) error {
	if f.value == "" {
		return nil
	}
	appendValue, appendFlagValue := f.selectAppenderFuncs(cli)
	if f.flag == "" {
		appendValue(f.value)
	} else {
		appendFlagValue(f.flag, f.value)
	}
	return nil
}

// selectAppenderFuncs returns the appropriate appender functions based on the redacted flag.
func (f stringFlag) selectAppenderFuncs(cli safecli.CommandAppender) (appenderFunc, appenderFunc) {
	if f.redacted {
		return cli.AppendRedacted, cli.AppendRedactedKV
	}
	return cli.AppendLoggable, cli.AppendLoggableKV
}

// newStringFlag creates a new string flag with a given flag name and value.
func newStringFlag(flag, val string, redacted bool) Applier {
	if flag == "" && val == "" {
		return ErrorFlag(cli.ErrInvalidFlag)
	}
	return stringFlag{flag: flag, value: val, redacted: redacted}
}

// NewStringFlag creates a new string flag with a given flag name and value.
func NewStringFlag(flag, val string) Applier {
	return newStringFlag(flag, val, false)
}

// NewRedactedStringFlag creates a new string flag with a given flag name and value.
func NewRedactedStringFlag(flag, val string) Applier {
	return newStringFlag(flag, val, true)
}

// NewStringArgument creates a new string argument with a given value.
func NewStringArgument(val string) Applier {
	return newStringFlag("", val, false)
}
