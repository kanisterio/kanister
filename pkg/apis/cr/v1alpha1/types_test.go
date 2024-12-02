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

package v1alpha1

import (
	"reflect"
	"testing"

	"github.com/kanisterio/errkit"
	"gopkg.in/check.v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

func Test(t *testing.T) { check.TestingT(t) }

type TypesSuite struct{}

var _ = check.Suite(&TypesSuite{})

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

func (s *TypesSuite) TestBlueprintDecode(c *check.C) {
	expected := map[string]reflect.Kind{
		"testint":         reflect.Int64,
		"teststring":      reflect.String,
		"teststringslice": reflect.Slice,
		"teststringmap":   reflect.Map,
	}

	bp, err := getBlueprintFromSpec([]byte(bpSpec))
	c.Assert(err, check.IsNil)
	c.Assert(bp.Actions["echo"].Phases[0].Args, check.HasLen, len(expected))
	for n, evk := range expected {
		v := bp.Actions["echo"].Phases[0].Args[n]
		c.Check(v, check.Not(check.Equals), nil)
		c.Check(reflect.TypeOf(v).Kind(), check.Equals, evk)
	}
}

// getBlueprintFromSpec returns a Blueprint object created from the given spec
func getBlueprintFromSpec(spec []byte) (*Blueprint, error) {
	blueprint := &Blueprint{}
	d := serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()
	if _, _, err := d.Decode([]byte(spec), nil, blueprint); err != nil {
		return nil, errkit.Wrap(err, "Failed to decode spec into object")
	}
	return blueprint, nil
}
