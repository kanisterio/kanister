package v1alpha1

import (
	"reflect"
	"testing"

	"github.com/pkg/errors"
	. "gopkg.in/check.v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

func Test(t *testing.T) { TestingT(t) }

type TypesSuite struct{}

var _ = Suite(&TypesSuite{})

const bpSpec = `
actions:
  echo:
    phases:
    - args:
        testint: 1
        teststring: "{{ .Deployment.Namespace }}"
        teststringslice:
            - postgresql
            - bash
            - -o
            - errexit
            - |
                env_dir="${PGDATA}/env"
                mkdir -p "${env_dir}"
                env_wal_prefix="${env_dir}/WALE_S3_PREFIX"
        teststringmap:
            foo: bar
`

func (s *TypesSuite) TestBlueprintDecode(c *C) {
	expected := map[string]reflect.Kind{
		"testint":         reflect.Int64,
		"teststring":      reflect.String,
		"teststringslice": reflect.Slice,
		"teststringmap":   reflect.Map,
	}

	bp, err := getBlueprintFromSpec([]byte(bpSpec))
	c.Assert(err, IsNil)
	c.Assert(bp.Actions["echo"].Phases[0].Args, HasLen, len(expected))
	for n, evk := range expected {
		v := bp.Actions["echo"].Phases[0].Args[n]
		c.Check(v, Not(Equals), nil)
		c.Check(reflect.TypeOf(v).Kind(), Equals, evk)
	}
}

// getBlueprintFromSpec returns a Blueprint object created from the given spec
func getBlueprintFromSpec(spec []byte) (*Blueprint, error) {
	blueprint := &Blueprint{}
	d := serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()
	if _, _, err := d.Decode([]byte(spec), nil, blueprint); err != nil {
		return nil, errors.Wrap(err, "Failed to decode spec into object")
	}
	return blueprint, nil
}
