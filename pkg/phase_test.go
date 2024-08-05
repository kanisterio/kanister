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

	. "gopkg.in/check.v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/utils"
)

type PhaseSuite struct{}

var (
	_      = Suite(&PhaseSuite{})
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

func (s *PhaseSuite) TestExec(c *C) {
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
		c.Assert(err, IsNil)
		p := Phase{args: args, f: tf}
		_, err = p.Exec(context.Background(), crv1alpha1.Blueprint{}, "", tp)
		c.Assert(err, IsNil)
		c.Assert(output, Equals, tc.expected)
	}
}

func (s *PhaseSuite) TestCheckSupportedArgs(c *C) {
	for _, tc := range []struct {
		supprtedArgs []string
		providedArgs map[string]interface{}
		err          Checker
		expErr       string
	}{
		{
			supprtedArgs: []string{"a", "b", "c"},
			providedArgs: map[string]interface{}{
				"a": "val",
				"b": "val",
				"c": "val",
			},
			err: IsNil,
		},
		{
			supprtedArgs: []string{"a", "b", "c"},
			providedArgs: map[string]interface{}{
				"a": "val",
				"b": "val",
				"c": "val",
				"d": "val",
			},
			err:    NotNil,
			expErr: "argument d is not supported",
		},
		{
			supprtedArgs: []string{"a", "b", "c"},
			providedArgs: map[string]interface{}{},
			err:          IsNil,
		},
	} {
		err := utils.CheckSupportedArgs(tc.supprtedArgs, tc.providedArgs)
		if err != nil {
			c.Assert(err.Error(), Equals, tc.expErr)
		}
		c.Assert(err, tc.err)
	}
}

func (s *PhaseSuite) TestRegFuncVersion(c *C) {
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
			c.Assert(err, IsNil)
		} else {
			err := RegisterVersion(tc.f, tc.regWithVersion)
			c.Assert(err, IsNil)
		}

		semVer, err := regFuncVersion(tc.f.Name(), tc.queryVersion)
		c.Assert(err, IsNil)
		c.Assert(semVer.Original(), Equals, tc.expectedVersion)
	}
}
