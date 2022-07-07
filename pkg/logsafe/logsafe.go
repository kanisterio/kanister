// Copyright 2022 The Kanister Authors.
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

// Package logsafe provides a framework for constructing commands that
// are safe for logging, where each token is considered unsafe
// and redacted, unless explicitly specified.
package logsafe

import (
	"strings"
)

const argRedacted = "<****>"

// Cmd is a way of building a command, token by token, such that
// it can be safely logged with redacted fields. The methods provided
// require the caller to explicitly specify whether a given token is
// safe to be logged or should be redacted.
type Cmd []arg

// NewLoggable initializes a safeCmd with the provided arguments,
// setting them all to safe to log.
func NewLoggable(args ...string) Cmd {
	return Cmd{}.AppendLoggable(args...)
}

// AppendRedacted adds one or more tokens to the command, all of which
// will be considered secrets and therefore redacted when the Stringer
// is called.
func (c Cmd) AppendRedacted(vl ...string) Cmd {
	for _, v := range vl {
		c = append(c, arg{value: v})
	}
	return c
}

// AppendRedactedKV appends a single key as safe to log, and its
// associated value as redacted.
func (c Cmd) AppendRedactedKV(k, v string) Cmd {
	c = append(c, arg{key: k, value: v})
	return c
}

// AppendLoggableKV appends a key-value pair of tokens to the
// safeCmd, both of which are considered safe to log.
func (c Cmd) AppendLoggableKV(k, v string) Cmd {
	c = append(c, arg{key: k, value: v, plainText: true})
	return c
}

// AppendLoggable appends a series of values as tokens
// to the command, all of which will be considered safe to log.
func (c Cmd) AppendLoggable(vl ...string) Cmd {
	for _, v := range vl {
		c = append(c, arg{value: v, plainText: true})
	}

	return c
}

// Combine two safeCmd structures into one by appending
// the tokens from the second into the first.
func (c Cmd) Combine(new Cmd) Cmd {
	return append(c, new...)
}

// String returns a loggable version of the safeCmd, obfuscating any fields
// that are not marked as loggable in plain text
func (c Cmd) String() string {
	argList := make([]string, len(c))
	for i, arg := range c {
		argList[i] = arg.String()
	}

	return strings.Join(argList, " ")
}

// PlainText returns the entire command as a slice of string arguments. All
// arguments will be returned in plain text.
func (c Cmd) PlainText() string {
	return strings.Join(c.combineValues(), " ")
}

func (c Cmd) combineValues() []string {
	if c == nil {
		return []string{}
	}

	argList := make([]string, len(c))
	for i, arg := range c {
		argList[i] = combineKeyValue(arg.key, arg.value)
	}
	return argList
}

// StringSliceCMD is going to return the command c in the string slice format unlike PlainText
func (c Cmd) StringSliceCMD() []string {
	return c.combineValues()
}

// Argv returns the argument vector
func (c Cmd) Argv() []string {
	if c == nil {
		return nil
	}

	argv := make([]string, 0, 2*len(c))

	for _, arg := range c {
		if arg.key != "" {
			argv = append(argv, arg.key)
		}
		argv = append(argv, arg.value)
	}

	return argv
}

// arg holds the value of a given token, and whether or not it is
// considered safe to log.
type arg struct {
	key       string
	value     string
	plainText bool
}

// String returns the value of the arg, or a redacted value if the arg
// is considered unsafe to log in plain text.
func (a arg) String() string {
	switch {
	case a.plainText:
		return combineKeyValue(a.key, a.value)
	default:
		return combineKeyValue(a.key, argRedacted)
	}
}

func combineKeyValue(k, v string) string {
	if k == "" {
		return v
	}
	return k + "=" + v
}
