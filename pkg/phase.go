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

	"github.com/Masterminds/semver"
	"github.com/pkg/errors"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/param"
)

// Phase is an atomic unit of execution.
type Phase struct {
	name    string
	args    map[string]interface{}
	objects map[string]crv1alpha1.ObjectReference
	f       Func
}

// Name returns the name of this phase.
func (p *Phase) Name() string {
	return p.name
}

// Objects returns the phase object references
func (p *Phase) Objects() map[string]crv1alpha1.ObjectReference {
	return p.objects
}

// Exec renders the argument templates in this Phase's Func and executes with
// those arguments.
func (p *Phase) Exec(ctx context.Context, bp crv1alpha1.Blueprint, action string, tp param.TemplateParams) (map[string]interface{}, error) {
	if p.args == nil {
		// Get the action from Blueprint
		a, ok := bp.Actions[action]
		if !ok {
			return nil, errors.Errorf("Action {%s} not found in action map", action)
		}
		// Render the argument templates for the Phase's function
		for _, ap := range a.Phases {
			if ap.Name != p.name {
				continue
			}
			args, err := param.RenderArgs(ap.Args, tp)
			if err != nil {
				return nil, err
			}
			if err = checkRequiredArgs(p.f.RequiredArgs(), args); err != nil {
				return nil, errors.Wrapf(err, "Required args missing for function %s", p.f.Name())
			}
			p.args = args
		}
	}
	// Execute the function
	return p.f.Exec(ctx, tp, p.args)
}

// GetPhases renders the returns a list of Phases with pre-rendered arguments.
func GetPhases(bp crv1alpha1.Blueprint, action, version string, tp param.TemplateParams) ([]*Phase, error) {
	a, ok := bp.Actions[action]
	if !ok {
		return nil, errors.Errorf("Action {%s} not found in action map", action)
	}
	defaultVersion, funcVersion, err := getFunctionVersion(version)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get function version")
	}
	funcMu.RLock()
	defer funcMu.RUnlock()
	phases := make([]*Phase, 0, len(a.Phases))
	// Check that all requested phases are registered and render object refs
	for _, p := range a.Phases {
		regVersion := *funcVersion
		if _, ok := funcs[p.Func]; !ok {
			return nil, errors.Errorf("Requested function {%s} has not been registered", p.Func)
		}
		if _, ok := funcs[p.Func][regVersion]; !ok {
			if funcVersion.Equal(defaultVersion) {
				return nil, errors.Errorf("Requested function {%s} has not been registered with version {%s}", p.Func, version)
			}
			if _, ok := funcs[p.Func][*defaultVersion]; !ok {
				return nil, errors.Errorf("Requested function {%s} has not been registered with versions {%s} or {%s}", p.Func, version, DefaultVersion)
			}
			log.Info().Print("Falling back to default version of the function", field.M{"Function": p.Func, "PreferredVersion": version, "FallbackVersion": DefaultVersion})
			regVersion = *defaultVersion
		}
		objs, err := param.RenderObjectRefs(p.ObjectRefs, tp)
		if err != nil {
			return nil, err
		}
		phases = append(phases, &Phase{
			name:    p.Name,
			objects: objs,
			f:       funcs[p.Func][regVersion],
			args:    p.Args,
		})
	}
	return phases, nil
}

func (p *Phase) Validate() error {
	return checkRequiredArgs(p.f.RequiredArgs(), p.args)
}

func checkRequiredArgs(reqArgs []string, args map[string]interface{}) error {
	for _, a := range reqArgs {
		if _, ok := args[a]; !ok {
			return errors.Errorf("Required arg missing: %s", a)
		}
	}
	return nil
}

func getFunctionVersion(version string) (*semver.Version, *semver.Version, error) {
	dv, err := semver.NewVersion(DefaultVersion)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to parse default function version")
	}
	switch version {
	case DefaultVersion, "":
		return dv, dv, nil
	default:
		fv, err := semver.NewVersion(version)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "Failed to parse function version {%s}", version)
		}
		return dv, fv, nil
	}
}
