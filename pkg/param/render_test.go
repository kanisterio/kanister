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
			arg: "{{ .Options.hello }}",
			tp: TemplateParams{
				Options: map[string]string{
					"hello": "",
				},
			},
			out:     "",
			checker: IsNil,
		},
		{
			arg: "{{ .Options.hello }}",
			tp: TemplateParams{
				Options: map[string]string{
					"hello": "someValue",
				},
			},
			out:     "someValue",
			checker: IsNil,
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
			checker: NotNil,
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
		{
			// Render should fail if referenced key doesn't exist
			arg: "{{ .Options.hello }}",
			tp: TemplateParams{
				Options: map[string]string{},
			},
			checker: NotNil,
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

func (s *RenderSuite) TestRenderObjects(c *C) {
	tp := TemplateParams{
		Object: map[string]interface{}{
			"spec": map[string]string{"authSecret": "secret-name"},
		},
	}
	in := map[string]crv1alpha1.ObjectReference{
		"authSecret": crv1alpha1.ObjectReference{
			Kind: SecretKind,
			Name: "{{ .Object.spec.authSecret }}",
		},
	}
	out, err := RenderObjectRefs(in, tp)
	c.Assert(err, IsNil)
	c.Assert(out["authSecret"].Name, Equals, "secret-name")
}
