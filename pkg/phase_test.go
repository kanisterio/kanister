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

func (tf *testFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) error {
	*tf.output = args["testKey"].(string)
	return tf.err
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
			argument: "{{ .ArtifactsOut.test.KeyValue.in }} world",
			expected: "hello world",
		},
		{
			artifact: "HELLO",
			argument: "{{ .ArtifactsOut.test.KeyValue.in | lower}} world",
			expected: "hello world",
		},
	} {
		var output string
		tf := &testFunc{output: &output}
		tp := param.TemplateParams{
			ArtifactsOut: map[string]crv1alpha1.Artifact{
				"test": crv1alpha1.Artifact{
					KeyValue: map[string]string{
						"in": tc.artifact,
					},
				},
			},
		}
		rawArgs := map[string]interface{}{
			"testKey": tc.argument,
		}
		args, err := param.RenderArgs(rawArgs, tp)
		c.Assert(err, IsNil)
		p := Phase{args: args, f: tf}
		err = p.Exec(context.Background(), tp)
		c.Assert(err, IsNil)
		c.Assert(output, Equals, tc.expected)
	}
}
