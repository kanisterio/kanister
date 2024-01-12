// Copyright 2019 The Kanister Authors.
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

package cmd

import "strings"

const (
	redactedValue     = "<****>"
	keyValueDelimiter = "="
)

// Argument stores a key=value pair where the value is subject to redaction.
type Argument struct {
	Key   string
	Value Redactor
}

// SensitiveValue implements Redactor interface for sensitive data.
type SensitiveValue struct {
	value string
}

// String returns a redacted string, never the actual value.
func (r SensitiveValue) String() string {
	return redactedValue
}

// GoString returns a redacted string, never the actual value for a %#v format too.
func (r SensitiveValue) GoString() string {
	return redactedValue
}

// PlainString returns the original sensitive value.
func (r SensitiveValue) PlainString() string {
	return r.value
}

// PlainValue implements Redactor interface for non-sensitive data.
type PlainValue struct {
	value string
}

// String returns a string as is, without redaction.
func (l PlainValue) String() string {
	return l.value
}

// PlainString returns a string as is.
func (l PlainValue) PlainString() string {
	return l.value
}

// Builder implements CommandBuilder and CommandData interfaces.
// It is used to build a command with a mix of loggable and redacted arguments.
type Builder struct {
	Args      []Argument
	Formatter ArgumentFormatter
}

// assert that Builder implements CommandBuilder and CommandArguments interfaces
// and Logger implements CommandLogger interface.
var (
	_ CommandBuilder   = (*Builder)(nil)
	_ CommandArguments = (*Builder)(nil)
	_ CommandLogger    = (*Logger)(nil)
)

// newPlain returns value as Plain.
func newPlain(value string) Redactor {
	return &PlainValue{value}
}

// newSensitive returns value as Sensitive.
func newSensitive(value string) Redactor {
	return &SensitiveValue{value}
}

// redactor is a function that converts value to Redactor.
type redactor func(value string) Redactor

// append adds a single argument to the command builder with a custom redactor.
func (b *Builder) append(key, value string, redactor redactor) {
	b.Args = append(b.Args, Argument{
		Key:   key,
		Value: redactor(value),
	})
}

// appendValues adds values to the command builder with a custom redactor.
func (b *Builder) appendValues(values []string, redactor redactor) *Builder {
	for _, arg := range values {
		b.append("", arg, redactor)
	}
	return b
}

// appendKeyValuePairs adds key=value pairs to the command builder with a custom redactor.
func (b *Builder) appendKeyValuePairs(kvPairs []string, redactor redactor) *Builder {
	for i := 0; i < len(kvPairs); i += 2 {
		key, value := kvPairs[i], ""
		if i+1 < len(kvPairs) {
			value = kvPairs[i+1]
		}
		b.append(key, value, redactor)
	}
	return b
}

// AppendLoggable adds loggable values to the command builder.
// These values can be logged and displayed as they do not have sensitive info.
func (b *Builder) AppendLoggable(values ...string) *Builder {
	return b.appendValues(values, newPlain)
}

// AppendRedacted adds redacted values to the command builder.
// These values are sensitive and should be display in logs as <****>.
func (b *Builder) AppendRedacted(values ...string) *Builder {
	return b.appendValues(values, newSensitive)
}

// AppendLoggableKV adds key=value pairs to the command builder.
// Key and value are loggable.
func (b *Builder) AppendLoggableKV(kvPairs ...string) *Builder {
	return b.appendKeyValuePairs(kvPairs, newPlain)
}

// AppendRedactedKV adds key=value pairs to the command builder.
// Key is loggable and value is sensitive.
// The value should be display in logs as <****>.
func (b *Builder) AppendRedactedKV(kvPairs ...string) *Builder {
	return b.appendKeyValuePairs(kvPairs, newSensitive)
}

// Append combines Builder with the command arguments.
func (b *Builder) Append(command CommandArguments) *Builder {
	b.Args = append(b.Args, command.Arguments()...)
	return b
}

// Arguments returns the command arguments.
func (b *Builder) Arguments() []Argument {
	return b.Args
}

// Build builds and returns the command.
func (b *Builder) Build() []string {
	return b.Formatter.format(b.Args)
}

// String returns a string representation of the Builder with hidden sensitive fields.
func (b *Builder) String() string {
	return NewLogger(b).Log()
}

// Logger is used for logging command arguments.
type Logger struct {
	command   CommandArguments
	Formatter ArgumentFormatter
}

// Log builds the loggable command string from the command arguments.
func (l *Logger) Log() string {
	c := l.Formatter.format(l.command.Arguments())
	return strings.Join(c, " ")
}
