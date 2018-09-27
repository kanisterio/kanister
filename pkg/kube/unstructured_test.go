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
