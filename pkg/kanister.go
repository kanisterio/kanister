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

package kanister

import (
	"context"
	"sync"

	"github.com/Masterminds/semver"
	"github.com/pkg/errors"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/param"
)

const (
	DefaultVersion = "v0.0.0"
)

var (
	funcMu sync.RWMutex
	funcs  = make(map[string]map[semver.Version]Func)
)

// Func allows custom actions to be executed.
type Func interface {
	Name() string
	RequiredArgs() []string
	Arguments() []string
	Exec(context.Context, param.TemplateParams, map[string]interface{}) (map[string]interface{}, error)
	ExecutionProgress() (crv1alpha1.PhaseProgress, error)
	Validate(map[string]any) error
}

// Register allows Funcs to be referenced by User Defined YAMLs
func Register(f Func) error {
	version := *semver.MustParse(DefaultVersion)
	funcMu.Lock()
	defer funcMu.Unlock()
	if f == nil {
		return errors.Errorf("kanister: Cannot register nil function")
	}
	if _, ok := funcs[f.Name()][version]; ok {
		panic("kanister: Register called twice for function " + f.Name() + " with version " + DefaultVersion)
	}
	if _, ok := funcs[f.Name()]; !ok {
		funcs[f.Name()] = make(map[semver.Version]Func)
	}
	funcs[f.Name()][version] = f
	return nil
}

func RegisteredFunctions() map[string]struct{} {
	names := make(map[string]struct{}, len(funcs))
	for k := range funcs {
		names[k] = struct{}{}
	}
	return names
}

func KanisterFuncForName(funcName, version string) Func {
	return funcs[funcName][*semver.MustParse(version)]
}

// RegisterVersion allows a Kanister Function to be registered with the given version
func RegisterVersion(f Func, v string) error {
	version := *semver.MustParse(v)
	funcMu.Lock()
	defer funcMu.Unlock()
	if f == nil {
		return errors.Errorf("kanister: Cannot register nil function")
	}
	if _, ok := funcs[f.Name()][version]; ok {
		panic("kanister: Register called twice for function " + f.Name() + " with version " + v)
	}
	if _, ok := funcs[f.Name()]; !ok {
		funcs[f.Name()] = make(map[semver.Version]Func)
	}
	funcs[f.Name()][version] = f
	return nil
}
