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
	"strings"
	"testing"

	. "gopkg.in/check.v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
)

func Test(t *testing.T) { TestingT(t) }

type ValidateBlueprint struct{}

var _ = Suite(&ValidateBlueprint{})

func (v *ValidateBlueprint) TestValidate(c *C) {
	for _, tc := range []struct {
		phases      []crv1alpha1.BlueprintPhase
		err         Checker
		errContains string
	}{
		{
			phases: []crv1alpha1.BlueprintPhase{
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
			phases: []crv1alpha1.BlueprintPhase{
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
			phases: []crv1alpha1.BlueprintPhase{
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
			phases: []crv1alpha1.BlueprintPhase{
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
			phases: []crv1alpha1.BlueprintPhase{
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
	} {
		bp := blueprint()
		bp.Actions["backup"].Phases = tc.phases
		err := Do(bp)
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
		},
	}
}
