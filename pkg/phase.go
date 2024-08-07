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
	"strings"

	"github.com/Masterminds/semver"
	"github.com/pkg/errors"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/utils"
)

var skipRenderFuncs = map[string]bool{
	"wait":   true,
	"waitv2": true,
}

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

// Progress return execution progress of the phase.
func (p *Phase) Progress() (crv1alpha1.PhaseProgress, error) {
	return p.f.ExecutionProgress()
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
		phases := []crv1alpha1.BlueprintPhase{}
		phases = append(phases, a.Phases...)
		if a.DeferPhase != nil {
			phases = append(phases, *a.DeferPhase)
		}

		err := p.setPhaseArgs(phases, tp)
		if err != nil {
			return nil, err
		}
	}
	// Execute the function
	return p.f.Exec(ctx, tp, p.args)
}

func (p *Phase) setPhaseArgs(phases []crv1alpha1.BlueprintPhase, tp param.TemplateParams) error {
	for _, ap := range phases {
		if ap.Name != p.name {
			continue
		}

		args, err := renderFuncArgs(ap.Func, ap.Args, tp)
		if err != nil {
			return err
		}

		if err = utils.CheckRequiredArgs(p.f.RequiredArgs(), args); err != nil {
			return errors.Wrapf(err, "Required args missing for function %s", p.f.Name())
		}

		if err = utils.CheckSupportedArgs(p.f.Arguments(), args); err != nil {
			return errors.Wrapf(err, "Checking supported args for function %s.", p.f.Name())
		}

		p.args = args
	}
	return nil
}

func renderFuncArgs(
	funcName string,
	args map[string]interface{},
	tp param.TemplateParams) (map[string]interface{}, error) {
	// let wait handle its own go template and jsonpath arguments
	if skipRenderFuncs[strings.ToLower(funcName)] {
		return args, nil
	}

	return param.RenderArgs(args, tp)
}

func GetDeferPhase(bp crv1alpha1.Blueprint, action, version string, tp param.TemplateParams) (*Phase, error) {
	a, ok := bp.Actions[action]
	if !ok {
		return nil, errors.Errorf("Action {%s} not found in blueprint actions", action)
	}

	if a.DeferPhase == nil {
		return nil, nil
	}

	regVersion, err := regFuncVersion(a.DeferPhase.Func, version)
	if err != nil {
		return nil, err
	}

	objs, err := param.RenderObjectRefs(a.DeferPhase.ObjectRefs, tp)
	if err != nil {
		return nil, err
	}

	return &Phase{
		name:    a.DeferPhase.Name,
		objects: objs,
		f:       funcs[a.DeferPhase.Func][regVersion],
	}, nil
}

func regFuncVersion(f, version string) (semver.Version, error) {
	funcMu.RLock()
	defer funcMu.RUnlock()

	defaultVersion, funcVersion, err := getFunctionVersion(version)
	if err != nil {
		return semver.Version{}, errors.Wrapf(err, "Failed to get function version")
	}

	regVersion := *funcVersion
	if _, ok := funcs[f]; !ok {
		return semver.Version{}, errors.Errorf("Requested function {%s} has not been registered", f)
	}
	if _, ok := funcs[f][regVersion]; !ok {
		if funcVersion.Equal(defaultVersion) {
			return semver.Version{}, errors.Errorf("Requested function {%s} has not been registered with version {%s}", f, version)
		}
		if _, ok := funcs[f][*defaultVersion]; !ok {
			return semver.Version{}, errors.Errorf("Requested function {%s} has not been registered with versions {%s} or {%s}", f, version, DefaultVersion)
		}
		log.Info().Print("Falling back to default version of the function", field.M{"Function": f, "PreferredVersion": version, "FallbackVersion": DefaultVersion})
		return *defaultVersion, nil
	}

	return *funcVersion, nil
}

// GetPhases renders the returns a list of Phases with pre-rendered arguments.
func GetPhases(bp crv1alpha1.Blueprint, action, version string, tp param.TemplateParams) ([]*Phase, error) {
	a, ok := bp.Actions[action]
	if !ok {
		return nil, errors.Errorf("Action {%s} not found in action map", action)
	}

	phases := make([]*Phase, 0, len(a.Phases))
	// Check that all requested phases are registered and render object refs
	for _, p := range a.Phases {
		regVersion, err := regFuncVersion(p.Func, version)
		if err != nil {
			return nil, err
		}

		objs, err := param.RenderObjectRefs(p.ObjectRefs, tp)
		if err != nil {
			return nil, err
		}
		phases = append(phases, &Phase{
			name:    p.Name,
			objects: objs,
			f:       funcs[p.Func][regVersion],
		})
	}
	return phases, nil
}

// Validate gets the provided arguments from a blueprint and calls Validate method of function to valdiate a function.
func (p *Phase) Validate(args map[string]any) error {
	return p.f.Validate(args)
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
