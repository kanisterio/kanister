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
	"errors"
	"fmt"

	"gopkg.in/check.v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/utils"
)

type PhaseSuite struct{}

var (
	_      = check.Suite(&PhaseSuite{})
	_ Func = (*testFunc)(nil)
)

type testFunc struct {
	output          *string
	err             error
	progressPercent string
}

type anotherFunc struct {
	testFunc
}

type conditionFunc struct {
	testFunc
}

func (*conditionFunc) Name() string {
	return "conditionTestFunc"
}

func (a *anotherFunc) Name() string {
	return "anotherTestFunc"
}

func (*testFunc) Name() string {
	return "mock"
}

func (tf *testFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	tf.progressPercent = "0"
	defer func() { tf.progressPercent = "100" }()

	*tf.output = args["testKey"].(string)
	return nil, tf.err
}

func (tf *testFunc) RequiredArgs() []string {
	return nil
}

func (tf *testFunc) Arguments() []string {
	return nil
}

func (tf *testFunc) Validate(args map[string]any) error {
	if err := utils.CheckSupportedArgs(tf.Arguments(), args); err != nil {
		return err
	}

	return utils.CheckRequiredArgs(tf.RequiredArgs(), args)
}

func (tf *testFunc) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	return crv1alpha1.PhaseProgress{ProgressPercent: tf.progressPercent}, nil
}

func (s *PhaseSuite) TestExec(c *check.C) {
	for _, tc := range []struct {
		artifact string
		argument string
		expected string
	}{
		{
			artifact: "hello",
			argument: "{{ .Options.test }} world",
			expected: "hello world",
		},
		{
			artifact: "HELLO",
			argument: "{{ .Options.test | lower}} world",
			expected: "hello world",
		},
	} {
		var output string
		tf := &testFunc{output: &output}
		tp := param.TemplateParams{
			Options: map[string]string{
				"test": tc.artifact,
			},
		}
		rawArgs := map[string]interface{}{
			"testKey": tc.argument,
		}
		args, err := param.RenderArgs(rawArgs, tp)
		c.Assert(err, check.IsNil)
		p := Phase{args: args, f: tf}
		_, err = p.Exec(context.Background(), crv1alpha1.Blueprint{}, "", tp)
		c.Assert(err, check.IsNil)
		c.Assert(output, check.Equals, tc.expected)
	}
}

func (s *PhaseSuite) TestCheckSupportedArgs(c *check.C) {
	for _, tc := range []struct {
		supprtedArgs []string
		providedArgs map[string]interface{}
		err          check.Checker
		expErr       string
	}{
		{
			supprtedArgs: []string{"a", "b", "c"},
			providedArgs: map[string]interface{}{
				"a": "val",
				"b": "val",
				"c": "val",
			},
			err: check.IsNil,
		},
		{
			supprtedArgs: []string{"a", "b", "c"},
			providedArgs: map[string]interface{}{
				"a": "val",
				"b": "val",
				"c": "val",
				"d": "val",
			},
			err:    check.NotNil,
			expErr: "argument d is not supported",
		},
		{
			supprtedArgs: []string{"a", "b", "c"},
			providedArgs: map[string]interface{}{},
			err:          check.IsNil,
		},
	} {
		err := utils.CheckSupportedArgs(tc.supprtedArgs, tc.providedArgs)
		if err != nil {
			c.Assert(err.Error(), check.Equals, tc.expErr)
		}
		c.Assert(err, tc.err)
	}
}

func (s *PhaseSuite) TestRegFuncVersion(c *check.C) {
	for _, tc := range []struct {
		regWithVersion  string
		expectedVersion string
		queryVersion    string
		f               Func
	}{
		{
			f:               &testFunc{},
			expectedVersion: "v0.0.0",
			queryVersion:    "v0.0.0",
		},
		{
			f:               &testFunc{},
			regWithVersion:  "v0.0.1",
			expectedVersion: "v0.0.1",
			queryVersion:    "v0.0.1",
		},
		{
			f:               &anotherFunc{},
			expectedVersion: "v0.0.0",
			queryVersion:    "v0.0.0",
		},
		{
			f:               &anotherFunc{},
			regWithVersion:  "v1.2.3",
			expectedVersion: "v1.2.3",
			queryVersion:    "v1.2.3",
		},
		{
			f:               &anotherFunc{},
			regWithVersion:  "v4.5.6",
			expectedVersion: "v4.5.6",
			queryVersion:    "v4.5.6",
		},
		{
			f:               &anotherFunc{},
			regWithVersion:  "v0.9.9",
			expectedVersion: "v0.0.0",
			// even though we are registering function version v0.9.9 we are querying the same function that is registered with version
			// v0.0.0 that is the reason we have v0.0.0 as expectedVersion
			queryVersion: "v0.0.0",
		},
		{
			f:               &anotherFunc{},
			regWithVersion:  "v0.1.1",
			expectedVersion: "v0.0.0",
			// since function anotherFunc is not registered with version v11.11.11, we will default to defaultFuncVersion (i.e., v0.0.0)
			// that is the reason the expected version hereis v0.0.0
			queryVersion: "v11.11.11",
		},
	} {
		if tc.regWithVersion == "" {
			err := Register(tc.f)
			c.Assert(err, check.IsNil)
		} else {
			err := RegisterVersion(tc.f, tc.regWithVersion)
			c.Assert(err, check.IsNil)
		}

		semVer, err := regFuncVersion(tc.f.Name(), tc.queryVersion)
		c.Assert(err, check.IsNil)
		c.Assert(semVer.Original(), check.Equals, tc.expectedVersion)
	}
}

func (s *PhaseSuite) TestPhaseCondition(c *check.C) {
	for _, tc := range []struct {
		artifact        string
		condition       string
		phases          map[string]*param.Phase
		expectedError   error
		argument        string
		expected        string
		expectedSkipped bool
	}{
		{
			// Incorrect phase condition throws error
			artifact:      "hello",
			argument:      "{{ .Options.test }} world",
			condition:     "{{",
			expectedError: ErrParsePhaseCondition,
			expected:      "",
		},
		{
			// Phase expression that doesn't result in either `true` or `false` throws error
			artifact:      "hello",
			argument:      "{{ .Options.test }} world",
			condition:     "{{\"test\"}}",
			expectedError: ErrPhaseExpressionNotConditional,
			expected:      "",
		},
		{
			artifact:      "hello",
			argument:      "{{ .Options.test }} world",
			condition:     "{{ .Invalid.Option }}",
			expectedError: ErrExecutePhaseCondition,
			expected:      "",
		},
		{
			// Empty phase expression results in running the phase function
			artifact:      "hello",
			argument:      "{{ .Options.test }} world",
			condition:     "",
			expectedError: nil,
			expected:      "hello world",
		},
		{
			// Phase function is executed when condition evaluates to `true`
			artifact:      "hello",
			argument:      "{{ .Options.test }} world",
			condition:     "{{ eq .Options.test \"hello\" }}",
			expectedError: nil,
			expected:      "hello world",
		},
		{
			// Phase function isn't executed when condition evaluates to `false`
			artifact:        "hello",
			argument:        "{{ .Options.test }} world",
			condition:       "{{ eq .Options.test \"HELLO\" }}",
			expectedError:   nil,
			expected:        "",
			expectedSkipped: true,
		},
		{
			// Result is case-insensitive and surrounding whitespace is ignored
			artifact:      "hello",
			argument:      "{{ .Options.test }} world",
			condition:     "  TRUE\n",
			expectedError: nil,
			expected:      "hello world",
		},
		{
			artifact:        "hello",
			argument:        "{{ .Options.test }} world",
			condition:       " False ",
			expectedError:   nil,
			expected:        "",
			expectedSkipped: true,
		},
		{
			// Condition can use outputs of previous phases
			artifact:  "hello",
			argument:  "{{ .Options.test }} world",
			condition: "{{ eq .Phases.p1.Output.x \"yes\" }}",
			phases: map[string]*param.Phase{
				"p1": {Output: map[string]interface{}{"x": "yes"}},
			},
			expectedError: nil,
			expected:      "hello world",
		},
		{
			artifact:  "hello",
			argument:  "{{ .Options.test }} world",
			condition: "{{ eq .Phases.p1.Output.x \"yes\" }}",
			phases: map[string]*param.Phase{
				"p1": {Output: map[string]interface{}{"x": "no"}},
			},
			expectedError:   nil,
			expected:        "",
			expectedSkipped: true,
		},
		{
			// Condition can combine multiple expressions
			artifact:      "hello",
			argument:      "{{ .Options.test }} world",
			condition:     "{{ and (eq .Options.test \"hello\") (not (eq .Options.test \"bye\")) }}",
			expectedError: nil,
			expected:      "hello world",
		},
	} {
		var output string
		tf := &testFunc{output: &output}
		tp := param.TemplateParams{
			Options: map[string]string{
				"test": tc.artifact,
			},
			Phases: tc.phases,
		}
		rawArgs := map[string]interface{}{
			"testKey": tc.argument,
		}
		args, err := param.RenderArgs(rawArgs, tp)
		c.Assert(err, check.IsNil)
		p := Phase{args: args, f: tf, condition: tc.condition}
		_, err = p.Exec(context.Background(), crv1alpha1.Blueprint{}, "", tp)
		if tc.expectedError != nil {
			c.Assert(errors.Is(err, tc.expectedError), check.Equals, true)
			c.Assert(p.PhaseSkipped(), check.Equals, false)
			continue
		}
		c.Assert(err, check.IsNil)
		c.Assert(output, check.Equals, tc.expected)
		c.Assert(p.PhaseSkipped(), check.Equals, tc.expectedSkipped)

		progress, err := p.Progress()
		c.Assert(err, check.IsNil)
		if tc.expectedSkipped {
			c.Assert(p.SkipReason(), check.Equals, fmt.Sprintf("if=`%s` evaluated to false.", tc.condition))
			c.Assert(progress.ProgressPercent, check.Equals, "100")
			c.Assert(progress.LastTransitionTime, check.NotNil)
			// The phase function must not have been invoked
			c.Assert(tf.progressPercent, check.Equals, "")
		} else {
			c.Assert(p.SkipReason(), check.Equals, "")
			// Progress is delegated to the phase function
			c.Assert(progress.ProgressPercent, check.Equals, tf.progressPercent)
			c.Assert(tf.progressPercent, check.Equals, "100")
		}
	}
}

func (s *PhaseSuite) TestPhaseValidateCondition(c *check.C) {
	for _, tc := range []struct {
		condition string
		checker   check.Checker
	}{
		{condition: "", checker: check.IsNil},
		{condition: "true", checker: check.IsNil},
		{condition: "{{ eq .Options.test \"hello\" }}", checker: check.IsNil},
		{condition: "{{", checker: check.NotNil},
		{condition: "{{ if }}", checker: check.NotNil},
		{condition: "{{ unknownFunc }}", checker: check.NotNil},
	} {
		var output string
		p := Phase{f: &testFunc{output: &output}, condition: tc.condition}
		c.Check(p.Validate(map[string]any{}), tc.checker, check.Commentf("condition: %q", tc.condition))
		c.Check(validatePhaseCondition(tc.condition), tc.checker, check.Commentf("condition: %q", tc.condition))
	}
}

func (s *PhaseSuite) TestGetPhasesCondition(c *check.C) {
	var output string
	err := Register(&conditionFunc{testFunc{output: &output}})
	c.Assert(err, check.IsNil)

	bp := crv1alpha1.Blueprint{
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"backup": {
				Phases: []crv1alpha1.BlueprintPhase{
					{Name: "noCondition", Func: "conditionTestFunc"},
					{Name: "withCondition", Func: "conditionTestFunc", If: "{{ eq .Options.test \"hello\" }}"},
				},
			},
		},
	}
	phases, err := GetPhases(bp, "backup", "v0.0.0", param.TemplateParams{})
	c.Assert(err, check.IsNil)
	c.Assert(phases, check.HasLen, 2)
	c.Assert(phases[0].condition, check.Equals, "")
	c.Assert(phases[1].condition, check.Equals, "{{ eq .Options.test \"hello\" }}")
	c.Assert(phases[0].PhaseSkipped(), check.Equals, false)
	c.Assert(phases[1].PhaseSkipped(), check.Equals, false)
}
