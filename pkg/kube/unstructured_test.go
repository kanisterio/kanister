// Copyright 2019 Kasten Inc.
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
	. "gopkg.in/check.v1"
	"text/template"

	"github.com/Masterminds/sprig"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type UnstructuredSuite struct{}

var _ = Suite(&UnstructuredSuite{})

type Param struct {
	Unstructured map[string]interface{}
}

func (s *UnstructuredSuite) TestFetch(c *C) {
	gvr := schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "serviceaccounts",
	}
	u, err := FetchUnstructuredObject(gvr, "default", "default")
	c.Assert(err, IsNil)

	buf := bytes.NewBuffer(nil)
	tp := Param{Unstructured: u.UnstructuredContent()}

	for _, tc := range []struct {
		arg string
	}{
		{"{{ .Unstructured.metadata.name }}"},
		{"{{ index .Unstructured.secrets 0 }}"},
	} {
		t, err := template.New("config").Option("missingkey=error").Funcs(sprig.TxtFuncMap()).Parse(tc.arg)
		c.Assert(err, IsNil)
		err = t.Execute(buf, tp)
		c.Assert(err, IsNil)

		c.Logf("Template: %s, Value: %s", tc.arg, buf.String())
	}
}
