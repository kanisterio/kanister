// Copyright 2019 Kasten Inc.
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

package kanister

import (
	"context"
	"sync"

	"github.com/pkg/errors"

	"github.com/kanisterio/kanister/pkg/param"
)

var (
	funcMu sync.RWMutex
	funcs  = make(map[string]Func)
)

// Func allows custom actions to be executed.
type Func interface {
	Name() string
	RequiredArgs() []string
	Exec(context.Context, param.TemplateParams, map[string]interface{}) (map[string]interface{}, error)
}

// Register allows Funcs to be references by User Defined YAMLs
func Register(f Func) error {
	funcMu.Lock()
	defer funcMu.Unlock()
	if f == nil {
		return errors.Errorf("kanister: Cannot register nil function")
	}
	if _, dup := funcs[f.Name()]; dup {
		panic("kanister: Register called twice for function " + f.Name())
	}
	funcs[f.Name()] = f
	return nil
}
