package param

import (
	. "gopkg.in/check.v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
)

type RenderSuite struct{}

var _ = Suite(&RenderSuite{})

func (s *RenderSuite) TestRender(c *C) {
	for _, tc := range []struct {
		arg     interface{}
		tp      TemplateParams
		out     interface{}
		checker Checker
	}{
		{
			arg:     "",
			tp:      TemplateParams{},
			out:     "",
			checker: IsNil,
		},
		{
			arg:     "hello",
			tp:      TemplateParams{},
			out:     "hello",
			checker: IsNil,
		},
		{
			arg:     "-",
			tp:      TemplateParams{},
			out:     "-",
			checker: IsNil,
		},
		{
			arg: "{{ .ArtifactsOut.hello.KeyValue }}",
			tp: TemplateParams{
				ArtifactsOut: map[string]crv1alpha1.Artifact{
					"hello": crv1alpha1.Artifact{},
				},
			},
			out:     "map[]",
			checker: IsNil,
		},
		{
			arg: "{{ .ArtifactsOut.hello.KeyValue.someKey }}",
			tp: TemplateParams{
				ArtifactsOut: map[string]crv1alpha1.Artifact{
					"hello": crv1alpha1.Artifact{
						KeyValue: map[string]string{
							"someKey": "someValue",
						},
					},
				},
			},
			out:     "someValue",
			checker: IsNil,
		},
		{
			// `-` cannot be used in a template path.
			arg: "{{ .ArtifactsOut.hello-world.KeyValue.someKey }}",
			tp: TemplateParams{
				ArtifactsOut: map[string]crv1alpha1.Artifact{
					"hello-world": crv1alpha1.Artifact{
						KeyValue: map[string]string{
							"someKey": "someValue",
						},
					},
				},
			},
			out:     "",
			checker: NotNil,
		},
		{
			// `-` can exist in artifact keys, it just cannot be used in path.
			arg: "{{ .ArtifactsOut.hello.KeyValue.someKey }}",
			tp: TemplateParams{
				ArtifactsOut: map[string]crv1alpha1.Artifact{
					"hello": crv1alpha1.Artifact{
						KeyValue: map[string]string{
							"someKey": "someValue",
						},
					},
					"hello-world": crv1alpha1.Artifact{
						KeyValue: map[string]string{
							"someKey": "someValue",
						},
					},
				},
			},
			out:     "someValue",
			checker: IsNil,
		},
		{
			arg:     "{{ upper `hello` }}",
			tp:      TemplateParams{},
			out:     "HELLO",
			checker: IsNil,
		},
		{
			arg:     []string{"{{ upper `hello` }}"},
			tp:      TemplateParams{},
			out:     []interface{}{"HELLO"},
			checker: IsNil,
		},
		{
			arg:     map[string]string{"name": "{{ upper `hello` }}"},
			tp:      TemplateParams{},
			out:     map[interface{}]interface{}{"name": "HELLO"},
			checker: IsNil,
		},
		{
			arg:     map[string][]string{"{{ upper `hello` }}": {"{{ upper `hello` }}"}},
			tp:      TemplateParams{},
			out:     map[interface{}]interface{}{"HELLO": []interface{}{"HELLO"}},
			checker: IsNil,
		},
	} {
		inArgs := map[string]interface{}{"arg": tc.arg}
		out, err := RenderArgs(inArgs, tc.tp)
		c.Assert(err, tc.checker)
		if err == nil {
			c.Assert(out["arg"], DeepEquals, tc.out)
		}
	}
}
