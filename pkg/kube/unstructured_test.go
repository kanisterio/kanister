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

package kube

import (
	"bytes"
	"context"
	"text/template"

	. "gopkg.in/check.v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kanisterio/kanister/pkg/ksprig"
)

type UnstructuredSuite struct{}

var _ = Suite(&UnstructuredSuite{})

type Param struct {
	Unstructured map[string]interface{}
}

func (s *UnstructuredSuite) TestFetch(c *C) {
	ctx := context.Background()
	gvr := schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "services",
	}
	u, err := FetchUnstructuredObject(ctx, gvr, "default", "kubernetes")
	c.Assert(err, IsNil)

	buf := bytes.NewBuffer(nil)
	tp := Param{Unstructured: u.UnstructuredContent()}

	for _, tc := range []struct {
		arg string
	}{
		{"{{ .Unstructured.metadata.name }}"},
		{"{{ .Unstructured.spec.clusterIP }}"},
	} {
		t, err := template.New("config").Option("missingkey=error").Funcs(ksprig.TxtFuncMap()).Parse(tc.arg)
		c.Assert(err, IsNil)
		err = t.Execute(buf, tp)
		c.Assert(err, IsNil)

		c.Logf("Template: %s, Value: %s", tc.arg, buf.String())
	}
}
