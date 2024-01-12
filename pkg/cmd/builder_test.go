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

import (
	"fmt"
	"log"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestPlainValue(t *testing.T) {
	l := PlainValue{"--arg"}
	if l.PlainString() != "--arg" {
		t.Errorf("Expected --arg, got %v", l.PlainString())
	}

	if l.String() != "--arg" {
		t.Errorf("Expected --arg, got %v", l.String())
	}
}

func TestSensitiveValue(t *testing.T) {
	r := SensitiveValue{"pass123"}
	if r.PlainString() != "pass123" {
		t.Errorf("Expected pass123, got %v", r.PlainString())
	}

	if r.String() != redactedValue {
		t.Errorf("Expected %v, got %v", redactedValue, r.String())
	}

	if r.GoString() != redactedValue {
		t.Errorf("Expected %v, got %v", redactedValue, r.String())
	}
}

func TestBuilderRedactedPrint(t *testing.T) {
	r := SensitiveValue{"pass123"}
	for _, tt := range []struct {
		fmt      string
		expected string
	}{
		{fmt: "", expected: redactedValue},
		{fmt: "%s", expected: redactedValue},
		{fmt: "%v", expected: redactedValue},
		{fmt: "%+v", expected: redactedValue},
		{fmt: "%q", expected: fmt.Sprintf("\"%s\"", redactedValue)},
		{fmt: "%#v", expected: redactedValue},
	} {
		var logOut strings.Builder
		logger := log.New(&logOut, "", 0)
		logger.SetOutput(&logOut)
		if tt.fmt == "" {
			logger.Print(r)
		} else {
			logger.Printf(tt.fmt, r)
		}

		got := strings.Trim(logOut.String(), "\n")
		if tt.expected != got {
			t.Errorf("%q: expected '%v', got '%v'", tt.fmt, tt.expected, got)
		}
	}
}

func TestBuilderAppendLoggable(t *testing.T) {
	expect := []string{"cmd", "--arg1"}

	builder := NewBuilder().
		AppendLoggable(expect...)

	if len(builder.Args) != 2 {
		t.Errorf("Expected %v args, got %v", len(expect), len(builder.Args))
	}

	for i, arg := range builder.Args {
		if _, ok := builder.Args[i].Value.(*PlainValue); !ok {
			t.Errorf("args[%v] to be of type cmd.Loggable, got %T", i, builder.Args[i].Value)
		}

		if expect[i] != arg.Value.PlainString() {
			t.Errorf("args[%v] expected %v, got %v", i, expect[i], arg.Value.PlainString())
		}
	}
}

func TestBuilderAppendRedacted(t *testing.T) {
	expect := []string{"--arg1", "--arg2"}

	builder := NewBuilder().
		AppendRedacted(expect...)

	if len(builder.Args) != 2 {
		t.Errorf("Expected %v args, got %v", len(expect), len(builder.Args))
	}

	for i, arg := range builder.Args {
		if _, ok := builder.Args[i].Value.(*SensitiveValue); !ok {
			t.Errorf("args[%v] to be of type cmd.Redacted, got %T", i, builder.Args[i].Value)
		}

		if redactedValue != arg.Value.String() {
			t.Errorf("args[%v] expected %v, got %v", i, redactedValue, arg.Value.String())
		}
	}
}

func TestBuilderAppendLoggableKV(t *testing.T) {
	builder := NewBuilder().
		AppendLoggableKV(
			"--temp-dir", "/tmp",
			"--log-dir", "/var/log",
			"--dry-run",
		)

	if len(builder.Args) != 3 {
		t.Errorf("Expected 3 args, got %v", len(builder.Args))
	}

	if builder.Args[0].Key != "--temp-dir" {
		t.Errorf("args[0].Key expected --temp-dir, got %v", builder.Args[0].Key)
	}
	if builder.Args[0].Value.PlainString() != "/tmp" {
		t.Errorf("args[0].Value().PlainString() expected /tmp, got %v", builder.Args[0].Value.PlainString())
	}
	if builder.Args[0].Value.String() != "/tmp" {
		t.Errorf("args[0].Value().String() expected /tmp, got %v", builder.Args[0].Value.String())
	}

	if builder.Args[1].Key != "--log-dir" {
		t.Errorf("args[1].Key expected --log-dir, got %v", builder.Args[1].Key)
	}
	if builder.Args[1].Value.PlainString() != "/var/log" {
		t.Errorf("args[1].Value().PlainString() expected /var/log, got %v", builder.Args[1].Value.PlainString())
	}
	if builder.Args[1].Value.String() != "/var/log" {
		t.Errorf("args[1].Value().String() expected /var/log, got %v", builder.Args[1].Value.String())
	}

	if builder.Args[2].Key != "--dry-run" {
		t.Errorf("args[2].Key expected --dry-run, got %v", builder.Args[2].Key)
	}
	if builder.Args[2].Value.PlainString() != "" {
		t.Errorf("args[2].Value().PlainString() expected \"\", got \"%v\"", builder.Args[2].Value.PlainString())
	}
	if builder.Args[2].Value.String() != "" {
		t.Errorf("args[2].Value().String() expected \"\", got \"%v\"", builder.Args[2].Value.String())
	}
}

func TestBuilderAppendRedactedKV(t *testing.T) {
	builder := NewBuilder().
		AppendRedactedKV(
			"--password", "pass123",
			"--dry-run",
		)

	if len(builder.Args) != 2 {
		t.Errorf("Expected 2 args, got %v", len(builder.Args))
	}

	if builder.Args[0].Key != "--password" {
		t.Errorf("args[0].Key expected --password, got %v", builder.Args[0].Key)
	}
	if builder.Args[0].Value.PlainString() != "pass123" {
		t.Errorf("args[0].Value.PlainString() expected pass123, got %v", builder.Args[0].Value.PlainString())
	}
	if builder.Args[0].Value.String() != redactedValue {
		t.Errorf("args[0].Value.PlainString() expected %v, got %v", redactedValue, builder.Args[0].Value.PlainString())
	}

	if builder.Args[1].Key != "--dry-run" {
		t.Errorf("args[1].Key expected --dry-run, got %v", builder.Args[1].Key)
	}
	if builder.Args[1].Value.PlainString() != "" {
		t.Errorf("args[1].Value().PlainString() expected \"\", got \"%v\"", builder.Args[1].Value.PlainString())
	}
	if builder.Args[1].Value.String() != redactedValue {
		t.Errorf("args[1].Value().String() expected \"%v\", got \"%v\"", redactedValue, builder.Args[1].Value.String())
	}
}

func TestBuilderAppend(t *testing.T) {
	expectedCmd := "cmd --temp-dir=/tmp --log-dir=/var/log subcmd --password=<****>"
	expectedLog := []string{
		"cmd",
		"--temp-dir=/tmp",
		"--log-dir=/var/log",
		"subcmd",
		"--password=pass123",
	}

	builder := NewBuilder().
		AppendLoggable("cmd").
		AppendLoggableKV("--temp-dir", "/tmp").
		AppendLoggableKV("--log-dir", "/var/log")

	subCmd := NewBuilder().
		AppendLoggable("subcmd").
		AppendRedactedKV("--password", "pass123")
	builder.Append(subCmd)

	if len(builder.Args) != 5 {
		t.Errorf("Expected 5 args, got %v", len(builder.Args))
	}

	gotCmd := builder.String()
	if expectedCmd != gotCmd {
		t.Errorf("Expected '%v', got '%v'", expectedCmd, gotCmd)
	}

	gotLog := builder.Build()
	if !reflect.DeepEqual(expectedLog, gotLog) {
		t.Errorf("Expected '%#v', got '%#v'", expectedLog, gotLog)
	}
}

func TestBuilderString(t *testing.T) {
	expected := "cmd --temp-dir=/tmp subcmd --password=<****>"

	builder := NewBuilder().
		AppendLoggable("cmd").
		AppendLoggableKV("--temp-dir", "/tmp").
		AppendLoggable("subcmd").
		AppendRedactedKV("--password", "pass123")

	for _, tt := range []struct {
		fmt      string
		expected string
	}{
		{fmt: "", expected: expected},
		{fmt: "%s", expected: expected},
		{fmt: "%v", expected: expected},
		{fmt: "%+v", expected: expected},
		{fmt: "%q", expected: fmt.Sprintf("\"%s\"", expected)},
		{fmt: "%#v", expected: `&cmd.Builder{Args:[]cmd.Argument{cmd.Argument{Key:"", Value:(*cmd.PlainValue)()}, cmd.Argument{Key:"--temp-dir", Value:(*cmd.PlainValue)()}, cmd.Argument{Key:"", Value:(*cmd.PlainValue)()}, cmd.Argument{Key:"--password", Value:<****>}}, Formatter:(cmd.ArgumentFormatter)()}`},
	} {
		var logOut strings.Builder
		logger := log.New(&logOut, "", 0)
		logger.SetOutput(&logOut)
		if tt.fmt == "" {
			logger.Print(builder)
		} else {
			logger.Printf(tt.fmt, builder)
		}

		got := strings.Trim(logOut.String(), "\n")
		got = removeHexNumbers(got)

		if tt.expected != got {
			t.Errorf("%q: \nexpected '%v', \ngot      '%v'", tt.fmt, tt.expected, got)
		}
	}
}

// removeHexNumbers removes hexadecimal (0x...) numbers from the string.
func removeHexNumbers(s string) string {
	regex := regexp.MustCompile(`0[xX][0-9a-fA-F]+`)
	return regex.ReplaceAllString(s, "")
}
