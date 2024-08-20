// Copyright 2022 The Kanister Authors.
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

package validate

import (
	"context"
	"strings"
	"testing"

	. "gopkg.in/check.v1"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/function"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/utils"
)

func Test(t *testing.T) { TestingT(t) }

type BlueprintTest struct {
	backupPhases  []crv1alpha1.BlueprintPhase
	err           Checker
	errContains   string
	deferPhase    *crv1alpha1.BlueprintPhase
	restorePhases []crv1alpha1.BlueprintPhase
}

const (
	nonDefaultFuncVersion = "v0.0.1"
)

type ValidateBlueprint struct{}

var _ = Suite(&ValidateBlueprint{})

func (v *ValidateBlueprint) TestValidate(c *C) {
	for _, tc := range []BlueprintTest{
		{
			backupPhases: []crv1alpha1.BlueprintPhase{
				{
					Func: "KubeTask",
					Name: "00",
					Args: map[string]interface{}{
						"image": "",
					},
				},
				{
					Func: "KubeExec",
					Name: "01",
					Args: map[string]interface{}{
						"namespace": "",
						"command":   "",
					},
				},
				{
					Func: "KubeExec",
					Name: "01",
					Args: map[string]interface{}{
						"namespace": "",
						"command":   "",
						"pod":       "",
					},
				},
			},
			errContains: "Required arg missing: command",
			err:         NotNil,
		},
		{
			backupPhases: []crv1alpha1.BlueprintPhase{
				{
					Func: "KubeTask",
					Name: "10",
					Args: map[string]interface{}{
						"image":   "",
						"command": "",
					},
				},
				{
					Func: "KubeExec",
					Name: "11",
					Args: map[string]interface{}{
						"namespace": "",
						"command":   "",
						"pod":       "",
					},
				},
			},
			err: IsNil,
		},
		{
			// function name is incorrect
			backupPhases: []crv1alpha1.BlueprintPhase{
				{
					Func: "KubeTasks",
					Name: "20",
					Args: map[string]interface{}{
						"image":   "",
						"command": "",
					},
				},
				{
					Func: "KubeExec",
					Name: "21",
					Args: map[string]interface{}{
						"namespace": "",
						"command":   "",
						"pod":       "",
					},
				},
			},
			errContains: "Requested function {KubeTasks} has not been registered",
			err:         NotNil,
		},
		{
			backupPhases: []crv1alpha1.BlueprintPhase{
				{
					Func: "PrepareData",
					Name: "30",
					Args: map[string]interface{}{
						"namespace": "",
						"image":     "",
						"command":   "",
					},
				},
			},
			err: IsNil,
		},
		{
			backupPhases: []crv1alpha1.BlueprintPhase{
				{
					Func: "PrepareData",
					Name: "40",
					Args: map[string]interface{}{
						"namespace": "",
						"image":     "",
					},
				},
			},
			errContains: "Required arg missing: command",
			err:         NotNil,
		},
		{
			backupPhases: []crv1alpha1.BlueprintPhase{
				{
					Func: "PrepareData",
					Name: "50",
					Args: map[string]interface{}{
						"namespace": "",
						"image":     "",
					},
				},
			},
			errContains: "Required arg missing: command",
			err:         NotNil,
			deferPhase: &crv1alpha1.BlueprintPhase{
				Func: "PrepareData",
				Name: "51",
				Args: map[string]interface{}{
					"namespace": "",
					"image":     "",
				},
			},
		},
		{
			backupPhases: []crv1alpha1.BlueprintPhase{
				{
					Func: "PrepareData",
					Name: "60",
					Args: map[string]interface{}{
						"namespace": "",
						"image":     "",
						"command":   "",
					},
				},
			},
			errContains: "Required arg missing: command",
			err:         NotNil,
			deferPhase: &crv1alpha1.BlueprintPhase{
				Func: "PrepareData",
				Name: "61",
				Args: map[string]interface{}{
					"namespace": "",
					"image":     "",
				},
			},
		},
		{
			backupPhases: []crv1alpha1.BlueprintPhase{
				{
					Func: "PrepareData",
					Name: "70",
					Args: map[string]interface{}{
						"namespace": "",
						"image":     "",
						"command":   "",
					},
				},
			},
			err: IsNil,
			deferPhase: &crv1alpha1.BlueprintPhase{
				Func: "PrepareData",
				Name: "71",
				Args: map[string]interface{}{
					"namespace": "",
					"image":     "",
					"command":   "",
				},
			},
		},
	} {
		bp := blueprint()
		bp.Actions["backup"].Phases = tc.backupPhases
		if tc.deferPhase != nil {
			bp.Actions["backup"].DeferPhase = tc.deferPhase
		}
		err := Do(bp, kanister.DefaultVersion)
		if err != nil {
			c.Assert(strings.Contains(err.Error(), tc.errContains), Equals, true)
		}
		c.Assert(err, tc.err)
	}
}

func (v *ValidateBlueprint) TestValidateNonDefaultVersion(c *C) {
	for _, tc := range []BlueprintTest{
		{
			backupPhases: []crv1alpha1.BlueprintPhase{
				{
					Func: "NonDefaultVersionFunc",
					Name: "00",
					Args: map[string]interface{}{
						"ndVersionArg0": "",
						"ndVersionArg1": "",
						"ndVersionArg2": "",
					},
				},
				{
					Func: "PrepareData",
					Name: "01",
					Args: map[string]interface{}{
						"namespace": "",
						"image":     "",
						"command":   "",
					},
				},
			},
			err: IsNil,
		},
		{
			// blueprint with one function that is registered with default version and
			// one function with non default version
			backupPhases: []crv1alpha1.BlueprintPhase{
				{
					Func: "NonDefaultVersionFunc",
					Name: "10",
					Args: map[string]interface{}{
						"ndVersionArg0":  "",
						"ndVersionArg1":  "",
						"ndVersionArg23": "",
					},
				},
				{
					Func: "PrepareData",
					Name: "11",
					Args: map[string]interface{}{
						"namespace": "",
						"image":     "",
						"command":   "",
					},
				},
			},
			err:         NotNil,
			errContains: "argument ndVersionArg23 is not supported",
		},
		{
			// blueprint where both the functions are registered with non default version
			backupPhases: []crv1alpha1.BlueprintPhase{
				{
					Func: "NonDefaultVersionFunc",
					Name: "20",
					Args: map[string]interface{}{
						"ndVersionArg0": "",
						"ndVersionArg1": "",
						"ndVersionArg2": "",
					},
				},
				{
					Func: "NonDefaultVersionFunc",
					Name: "21",
					Args: map[string]interface{}{
						"ndVersionArg0": "",
						"ndVersionArg1": "",
					},
				},
			},
			err:         NotNil,
			errContains: "Required arg missing: ndVersionArg2",
		},
		{
			// blueprint where both the functions are registered with default version
			backupPhases: []crv1alpha1.BlueprintPhase{
				{
					Func: "PrepareData",
					Name: "30",
					Args: map[string]interface{}{
						"namespace": "",
						"image":     "",
						"command":   "",
					},
				},
				{
					Func: "PrepareData",
					Name: "31",
					Args: map[string]interface{}{
						"namespace": "",
						"image":     "",
						"command":   "",
					},
				},
			},
			err: IsNil,
		},
	} {
		bp := blueprint()
		bp.Actions["backup"].Phases = tc.backupPhases
		err := Do(bp, nonDefaultFuncVersion)
		if err != nil {
			c.Assert(strings.Contains(err.Error(), tc.errContains), Equals, true)
		}
		c.Assert(err, tc.err)
	}
}

func (v *ValidateBlueprint) TestValidateAnnLabelArgs(c *C) {
	for _, tc := range []struct {
		labels      interface{}
		annotations interface{}
		error       string
	}{
		{
			labels: map[string]interface{}{
				"key": "value",
			},
			error: "",
		},
		{
			annotations: map[string]interface{}{
				"key": "value",
			},
			error: "",
		},
		{
			labels: map[string]interface{}{
				"key": "value",
			},
			annotations: map[string]interface{}{
				"key": "value",
			},
			error: "",
		},
		{
			labels: map[string]interface{}{
				"key$": "value",
			},
			annotations: map[string]interface{}{
				"key": "value",
			},
			error: "label key 'key$' failed validation",
		},
		{
			labels: map[string]interface{}{
				"key*": "value",
			},
			annotations: map[string]interface{}{
				"key": "value",
			},
			error: "label key 'key*' failed validation",
		},
		{
			labels: map[string]interface{}{
				"key": "value$",
			},
			annotations: map[string]interface{}{
				"key": "value",
			},
			error: "label value 'value$' failed validation",
		},
		{
			labels: map[string]interface{}{
				"key": "value",
			},
			annotations: map[string]interface{}{
				"key$": "value",
			},
			error: "annotation key 'key$' failed validation",
		},
		{
			labels: map[string]interface{}{
				"key": "value",
			},
			annotations: map[string]interface{}{
				"key": "value$",
			},
			error: "",
		},
		{
			labels: map[string]interface{}{
				"key": "",
			},
			annotations: map[string]interface{}{
				"key": "",
			},
			error: "",
		},
	} {
		bp := blueprint()
		bp.Actions["backup"].Phases = []crv1alpha1.BlueprintPhase{
			{
				Func: "KubeTask",
				Name: "backup",
				Args: map[string]interface{}{
					function.PodLabelsArg:      tc.labels,
					function.PodAnnotationsArg: tc.annotations,
					"image":                    "",
					"command":                  "",
				},
			},
		}
		err := Do(bp, kanister.DefaultVersion)
		if tc.error != "" {
			c.Assert(strings.Contains(err.Error(), tc.error), Equals, true)
		} else {
			c.Assert(err, Equals, nil)
		}
	}
}

func (v *ValidateBlueprint) TestValidatePhaseNames(c *C) {
	for _, tc := range []BlueprintTest{
		{
			backupPhases: []crv1alpha1.BlueprintPhase{
				{Name: "phaseone"},
				{Name: "phasetwo"},
				{Name: "phasethree"},
			},
			restorePhases: []crv1alpha1.BlueprintPhase{
				{Name: "phasefour"},
				{Name: "phasefive"},
			},
			err: IsNil,
			deferPhase: &crv1alpha1.BlueprintPhase{
				Name: "phasesix",
			},
		},
		// duplicate phase names in the same action
		{
			backupPhases: []crv1alpha1.BlueprintPhase{
				{Name: "phaseone"},
				{Name: "phaseone"},
				{Name: "phasethree"},
			},
			restorePhases: []crv1alpha1.BlueprintPhase{
				{Name: "phasefour"},
				{Name: "phasefive"},
			},
			err:         NotNil,
			errContains: "Duplicated phase name is not allowed. Violating phase 'phaseone'",
			deferPhase: &crv1alpha1.BlueprintPhase{
				Name: "phasesix",
			},
		},
		// duplicate phase names in different actions
		{
			backupPhases: []crv1alpha1.BlueprintPhase{
				{Name: "phaseone"},
				{Name: "phasetwo"},
				{Name: "phasethree"},
			},
			restorePhases: []crv1alpha1.BlueprintPhase{
				{Name: "phaseone"},
				{Name: "phasefive"},
			},
			err:         NotNil,
			errContains: "Duplicated phase name is not allowed. Violating phase 'phaseone'",
			deferPhase: &crv1alpha1.BlueprintPhase{
				Name: "phasesix",
			},
		},
		// duplicate phase names in main phase and deferPhase
		{
			backupPhases: []crv1alpha1.BlueprintPhase{
				{Name: "phaseone"},
				{Name: "phasetwo"},
				{Name: "phasethree"},
			},
			restorePhases: []crv1alpha1.BlueprintPhase{
				{Name: "phasefour"},
				{Name: "phasefive"},
			},
			err:         NotNil,
			errContains: "Duplicated phase name is not allowed. Violating phase 'phaseone'",
			deferPhase: &crv1alpha1.BlueprintPhase{
				Name: "phaseone",
			},
		},
	} {
		bp := blueprint()
		bp.Actions["backup"].Phases = tc.backupPhases
		bp.Actions["restore"].Phases = tc.restorePhases
		if tc.deferPhase != nil {
			bp.Actions["backup"].DeferPhase = tc.deferPhase
		}
		err := validatePhaseNames(bp)
		if err != nil {
			c.Assert(strings.Contains(err.Error(), tc.errContains), Equals, true)
		}
		c.Assert(err, tc.err)
	}
}

func blueprint() *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"backup": {
				Phases: []crv1alpha1.BlueprintPhase{},
			},
			"restore": {
				Phases: []crv1alpha1.BlueprintPhase{},
			},
		},
	}
}

type nonDefaultVersionFunc struct {
	progressPercent string
}

func (nd *nonDefaultVersionFunc) Name() string {
	return "NonDefaultVersionFunc"
}

func (nd *nonDefaultVersionFunc) RequiredArgs() []string {
	return []string{"ndVersionArg0", "ndVersionArg1", "ndVersionArg2"}
}

func (nd *nonDefaultVersionFunc) Arguments() []string {
	return []string{"ndVersionArg0", "ndVersionArg1", "ndVersionArg2", "ndVersionArg3"}
}

func (nd *nonDefaultVersionFunc) Validate(args map[string]any) error {
	if err := utils.CheckSupportedArgs(nd.Arguments(), args); err != nil {
		return err
	}

	return utils.CheckRequiredArgs(nd.RequiredArgs(), args)
}

func (nd *nonDefaultVersionFunc) Exec(context.Context, param.TemplateParams, map[string]interface{}) (map[string]interface{}, error) {
	nd.progressPercent = "0"
	defer func() { nd.progressPercent = "100" }()
	return nil, nil
}

func (nd *nonDefaultVersionFunc) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	return crv1alpha1.PhaseProgress{ProgressPercent: nd.progressPercent}, nil
}

var _ kanister.Func = (*nonDefaultVersionFunc)(nil)

func init() {
	_ = kanister.RegisterVersion(&nonDefaultVersionFunc{}, nonDefaultFuncVersion)
}
