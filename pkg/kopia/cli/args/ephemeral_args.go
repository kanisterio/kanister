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

package args

import (
	"fmt"

	"github.com/kanisterio/safecli/command"

	"github.com/kanisterio/kanister/pkg/logsafe"
)

var (
	RepositoryCreate        Args
	RepositoryConnectServer Args
	UserAddSet              Args
)

type Args struct {
	args map[string]string
}

func (a *Args) Set(key, value string) {
	if a.args == nil {
		a.args = make(map[string]string)
	}
	if _, ok := a.args[key]; ok {
		panic(fmt.Sprintf("key %q already registered", key))
	}
	a.args[key] = value
}

func (a *Args) AppendToCmd(cmd logsafe.Cmd) logsafe.Cmd {
	for k, v := range a.args {
		cmd = cmd.AppendLoggableKV(k, v)
	}
	return cmd
}

func (a *Args) CommandAppliers() []command.Applier {
	appliers := make([]command.Applier, len(a.args))
	for k, v := range a.args {
		appliers = append(appliers, command.NewOptionWithArgument(k, v))
	}
	return appliers
}
