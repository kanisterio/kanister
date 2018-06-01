package param

import (
	"bytes"
	"reflect"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/pkg/errors"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
)

// RenderArgs function renders the arguments required for execution
func RenderArgs(args map[string]interface{}, tp TemplateParams) (map[string]interface{}, error) {
	ram := make(map[string]interface{}, len(args))
	for n, v := range args {
		rv, err := render(v, tp)
		if err != nil {
			return nil, err
		}
		ram[n] = rv
	}
	return ram, nil
}

// render will recurse through all args and render any strings
func render(arg interface{}, tp TemplateParams) (interface{}, error) {
	val := reflect.ValueOf(arg)
	switch reflect.TypeOf(arg).Kind() {
	case reflect.String:
		return renderStringArg(val.String(), tp)
	case reflect.Slice:
		ras := make([]interface{}, 0, val.Len())
		for i := 0; i < val.Len(); i++ {
			r, err := render(val.Index(i).Interface(), tp)
			if err != nil {
				return nil, err
			}
			ras = append(ras, r)
		}
		return ras, nil
	case reflect.Map:
		ras := make(map[interface{}]interface{}, val.Len())
		for _, k := range val.MapKeys() {
			rv, err := render(val.MapIndex(k).Interface(), tp)
			if err != nil {
				return nil, err
			}
			ras[k.Interface()] = rv
		}
		return ras, nil
	default:
		return arg, nil
	}
}

// RenderArtifacts function renders the artifacts required for execution
func RenderArtifacts(arts map[string]crv1alpha1.Artifact, tp TemplateParams) (map[string]crv1alpha1.Artifact, error) {
	rarts := make(map[string]crv1alpha1.Artifact, len(arts))
	for name, a := range arts {
		ra := crv1alpha1.Artifact{}
		for k, v := range a.KeyValue {
			rv, err := renderStringArg(v, tp)
			if err != nil {
				return nil, err
			}
			if ra.KeyValue == nil {
				ra.KeyValue = make(map[string]string, len(a.KeyValue))
			}
			ra.KeyValue[k] = rv
		}
		rarts[name] = ra
	}
	return rarts, nil
}

func renderStringArg(arg string, tp TemplateParams) (string, error) {
	t, err := template.New("config").Funcs(sprig.TxtFuncMap()).Parse(arg)
	if err != nil {
		return "", errors.WithStack(err)
	}
	buf := bytes.NewBuffer(nil)
	if err = t.Execute(buf, tp); err != nil {
		return "", errors.WithStack(err)
	}
	return buf.String(), nil
}
