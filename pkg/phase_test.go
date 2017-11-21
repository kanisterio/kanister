package kanister

import (
	"context"
	"strings"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	. "gopkg.in/check.v1"
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

func (tf *testFunc) Exec(ctx context.Context, args ...string) error {
	*tf.output = strings.Join(args, " ")
	return tf.err
}

func (s *PhaseSuite) TestExec(c *C) {
	var output string
	tf := &testFunc{output: &output}
	tp := TemplateParams{
		ArtifactsOut: map[string]crv1alpha1.Artifact{"test": crv1alpha1.Artifact{"in": "hello"}},
	}
	args, err := renderArgs([]string{"{{ .ArtifactsOut.test.in }}", "world"}, tp)
	p := Phase{
		args: args,
		f:    tf,
	}
	err = p.Exec(context.Background())
	c.Assert(err, IsNil)
	c.Assert(output, Equals, "hello world")
}

func (s *PhaseSuite) TestExecSprig(c *C) {
	var output string
	tf := &testFunc{output: &output}
	tp := TemplateParams{
		ArtifactsOut: map[string]crv1alpha1.Artifact{"test": crv1alpha1.Artifact{"in": "hello"}},
	}
	args, err := renderArgs([]string{"{{ .ArtifactsOut.test.in | lower}}", "world"}, tp)
	p := Phase{
		args: args,
		f:    tf,
	}
	err = p.Exec(context.Background())
	c.Assert(err, IsNil)
	// If the Sprig template functions work as expected, the HELLO argument to
	// the phase should be converted to lower case.
	c.Assert(output, Equals, "hello world")
}

func (s *PhaseSuite) TestRenderStrings(c *C) {
	for _, tc := range []struct {
		arg     string
		tp      TemplateParams
		out     string
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
			out:     "",
			checker: IsNil,
		},
		{
			arg: "{{ .ArtifactsOut.hello }}",
			tp: TemplateParams{
				ArtifactsOut: map[string]crv1alpha1.Artifact{
					"hello": crv1alpha1.Artifact{},
				},
			},
			out:     "",
			checker: IsNil,
		},
		{
			arg: "{{ .ArtifactsOut.hello.someKey }}",
			tp: TemplateParams{
				ArtifactsOut: map[string]crv1alpha1.Artifact{
					"hello": crv1alpha1.Artifact{"someKey": "someValue"},
				},
			},
			out:     "someValue",
			checker: IsNil,
		},
		{
			// `-` cannot be used in a template path.
			arg: "{{ .ArtifactsOut.hello-world.someKey }}",
			tp: TemplateParams{
				ArtifactsOut: map[string]crv1alpha1.Artifact{
					"hello-world": crv1alpha1.Artifact{"someKey": "someValue"},
				},
			},
			out:     "",
			checker: NotNil,
		},
		{
			// `-` can exist in artifact keys, it just cannot be used in path.
			arg: "{{ .ArtifactsOut.hello.someKey }}",
			tp: TemplateParams{
				ArtifactsOut: map[string]crv1alpha1.Artifact{
					"hello":       crv1alpha1.Artifact{"someKey": "someValue"},
					"hello-world": crv1alpha1.Artifact{"someKey": "someValue"},
				},
			},
			out:     "someValue",
			checker: IsNil,
		},
	} {
		out, err := renderString(tc.arg, tc.tp)
		c.Assert(err, tc.checker)
		c.Assert(out, Equals, out)
	}
}
