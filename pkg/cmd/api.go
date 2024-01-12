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

// Redactor defines an interface for handling sensitive value in the way
// that it can be represented both in a plain and redacted text form.
type Redactor interface {
	PlainString() string
	String() string
}

// CommandBuilder builds and returns the command for execution.
type CommandBuilder interface {
	Build() []string
}

// CommandLogger returns a string representation of the command for logging.
type CommandLogger interface {
	Log() string
}

// NewBuilder creates a new command builder instance.
func NewBuilder() *Builder {
	return &Builder{
		Formatter: CommandArgumentsFormatter,
	}
}

// CommandArguments provides an interface for accessing command arguments.
type CommandArguments interface {
	Arguments() []Argument
}

// NewLogger creates a new Logger instance.
func NewLogger(args CommandArguments) *Logger {
	return &Logger{
		args:      args,
		Formatter: LogArgumentsFormatter,
	}
}
