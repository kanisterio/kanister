package kanister

import (
	"context"
	"strings"

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

func (tf *testFunc) Exec(ctx context.Context, args ...string) error {
	*tf.output = strings.Join(args, " ")
	return tf.err
}

func (s *PhaseSuite) TestExec(c *C) {
	var output string
	tf := &testFunc{output: &output}
	tp := param.TemplateParams{
		ArtifactsOut: map[string]crv1alpha1.Artifact{"test": crv1alpha1.Artifact{KeyValue: map[string]string{"in": "hello"}}},
	}
	args, err := param.RenderArgs([]string{"{{ .ArtifactsOut.test.KeyValue.in }}", "world"}, tp)
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
	tp := param.TemplateParams{
		ArtifactsOut: map[string]crv1alpha1.Artifact{"test": crv1alpha1.Artifact{KeyValue: map[string]string{"in": "hello"}}},
	}
	args, err := param.RenderArgs([]string{"{{ .ArtifactsOut.test.KeyValue.in | lower}}", "world"}, tp)
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
