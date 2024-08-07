// Copyright 2023 The Kanister Authors.
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

package kube

import (
	"context"
	"io"
)

type FakePodCommandExecutor struct {
	ExecErr       error
	inExecCommand []string

	ExecStdout string
	ExecStderr string
}

func (fce *FakePodCommandExecutor) Exec(_ context.Context, command []string, stdin io.Reader, stdout, stderr io.Writer) error {
	fce.inExecCommand = make([]string, len(command))
	copy(fce.inExecCommand, command)
	if stdout != nil && len(fce.ExecStdout) > 0 {
		stdout.Write([]byte(fce.ExecStdout)) //nolint:errcheck
	}
	if stderr != nil && len(fce.ExecStderr) > 0 {
		stderr.Write([]byte(fce.ExecStderr)) //nolint:errcheck
	}

	return fce.ExecErr
}
