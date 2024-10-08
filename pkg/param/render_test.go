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

package param

import (
	"time"

	"gopkg.in/check.v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
)

type RenderSuite struct{}

var _ = check.Suite(&RenderSuite{})

func (s *RenderSuite) TestRender(c *check.C) {
	for _, tc := range []struct {
		arg     interface{}
		tp      TemplateParams
		out     interface{}
		checker check.Checker
	}{
		{
			arg:     "",
			tp:      TemplateParams{},
			out:     "",
			checker: check.IsNil,
		},
		{
			arg:     "hello",
			tp:      TemplateParams{},
			out:     "hello",
			checker: check.IsNil,
		},
		{
			arg:     "-",
			tp:      TemplateParams{},
			out:     "-",
			checker: check.IsNil,
		},
		{
			arg: "{{ .Options.hello }}",
			tp: TemplateParams{
				Options: map[string]string{
					"hello": "",
				},
			},
			out:     "",
			checker: check.IsNil,
		},
		{
			arg: "{{ .Options.hello }}",
			tp: TemplateParams{
				Options: map[string]string{
					"hello": "someValue",
				},
			},
			out:     "someValue",
			checker: check.IsNil,
		},
		{
			// `-` cannot be used in a template path.
			arg: "{{ .Options.hello-world }}",
			tp: TemplateParams{
				Options: map[string]string{
					"hello-world": "someValue",
				},
			},
			out:     "",
			checker: check.NotNil,
		},
		{
			// `-` can exist in artifact keys, it just cannot be used in path.
			arg: "{{ .Options.hello }}",
			tp: TemplateParams{
				Options: map[string]string{
					"hello":       "someValue",
					"hello-world": "someValue",
				},
			},
			out:     "someValue",
			checker: check.IsNil,
		},
		{
			arg:     "{{ upper `hello` }}",
			tp:      TemplateParams{},
			out:     "HELLO",
			checker: check.IsNil,
		},
		{
			arg:     []string{"{{ upper `hello` }}"},
			tp:      TemplateParams{},
			out:     []interface{}{"HELLO"},
			checker: check.IsNil,
		},
		{
			arg:     map[string]string{"name": "{{ upper `hello` }}"},
			tp:      TemplateParams{},
			out:     map[interface{}]interface{}{"name": "HELLO"},
			checker: check.IsNil,
		},
		{
			arg:     map[string][]string{"{{ upper `hello` }}": {"{{ upper `hello` }}"}},
			tp:      TemplateParams{},
			out:     map[interface{}]interface{}{"HELLO": []interface{}{"HELLO"}},
			checker: check.IsNil,
		},
		{
			// Render should fail if referenced key doesn't exist
			arg: "{{ .Options.hello }}",
			tp: TemplateParams{
				Options: map[string]string{},
			},
			checker: check.NotNil,
		},
	} {
		inArgs := map[string]interface{}{"arg": tc.arg}
		out, err := RenderArgs(inArgs, tc.tp)
		c.Assert(err, tc.checker)
		if err == nil {
			c.Assert(out["arg"], check.DeepEquals, tc.out)
		}
	}
}

func (s *RenderSuite) TestRenderObjects(c *check.C) {
	tp := TemplateParams{
		Time: time.Now().String(),
		Object: map[string]interface{}{
			"spec": map[string]string{"authSecret": "secret-name"},
		},
	}
	in := map[string]crv1alpha1.ObjectReference{
		"authSecret": {
			Kind: SecretKind,
			Name: "{{ .Object.spec.authSecret }}",
		},
	}
	out, err := RenderObjectRefs(in, tp)
	c.Assert(err, check.IsNil)
	c.Assert(out["authSecret"].Name, check.Equals, "secret-name")
}

func (s *RenderSuite) TestRenderArtifacts(c *check.C) {
	tp := TemplateParams{
		Phases: map[string]*Phase{
			"myPhase": {
				Output: map[string]interface{}{
					"kopiaSnapshot": "a-snapshot-id",
				},
			},
		},
	}

	for _, tc := range []struct {
		art     map[string]crv1alpha1.Artifact
		tp      TemplateParams
		out     map[string]crv1alpha1.Artifact
		checker check.Checker
	}{
		{
			art: map[string]crv1alpha1.Artifact{
				"myArt": {
					KopiaSnapshot: "{{ .Phases.myPhase.Output.kopiaSnapshot }}",
				},
			},
			tp: tp,
			out: map[string]crv1alpha1.Artifact{
				"myArt": {
					KopiaSnapshot: "a-snapshot-id",
				},
			},
			checker: check.IsNil,
		},

		{
			art: map[string]crv1alpha1.Artifact{
				"myArt": {
					KeyValue: map[string]string{
						"key": "{{ .Phases.myPhase.Output.kopiaSnapshot }}",
					},
				},
			},
			tp: tp,
			out: map[string]crv1alpha1.Artifact{
				"myArt": {
					KeyValue: map[string]string{
						"key": "a-snapshot-id",
					},
				},
			},
			checker: check.IsNil,
		},
	} {
		ra, err := RenderArtifacts(tc.art, tc.tp)
		c.Assert(err, tc.checker)
		c.Assert(ra, check.DeepEquals, tc.out)
	}
}
