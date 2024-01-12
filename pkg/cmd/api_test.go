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
	"reflect"
	"testing"
)

func createTestCommand() *Builder {
	return NewBuilder().
		AppendLoggable("cmd").
		AppendLoggableKV("--temp-dir", "/tmp").
		Append(createTestSubCommand())
}

func createTestSubCommand() *Builder {
	return NewBuilder().
		AppendLoggable("subcmd").
		AppendRedactedKV("--password", "pass123")
}

func TestNewCommandBuilder(t *testing.T) {
	builder := createTestCommand()

	expect := []string{"cmd", "--temp-dir=/tmp", "subcmd", "--password=pass123"}
	got := builder.Build()
	if len(expect) != len(got) {
		t.Errorf("Expected %v args, got %v", len(expect), len(got))
	}

	if !reflect.DeepEqual(expect, got) {
		t.Errorf("Expected %v, got %v", expect, got)
	}
}

func TestNewCommandLogger(t *testing.T) {
	logger := NewLogger(createTestCommand())

	expect := fmt.Sprintf("cmd --temp-dir=/tmp subcmd --password=%v", redactedValue)
	got := logger.Log()
	if len(expect) != len(got) {
		t.Errorf("Expected %v args, got %v", len(expect), len(got))
	}

	if !reflect.DeepEqual(expect, got) {
		t.Errorf("Expected %v, got %v", expect, got)
	}
}
