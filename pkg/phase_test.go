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
)

type PhaseSuite struct{}

var (
	_      = Suite(&PhaseSuite{})
	_ Func = (*testFunc)(nil)
)

type testFunc struct {
	output *string
	err    error
}

func (*testFunc) Name() string {
	return "mock"
}

func (tf *testFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	*tf.output = args["testKey"].(string)
	return nil, tf.err
}

func (tf *testFunc) RequiredArgs() []string {
	return nil
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
