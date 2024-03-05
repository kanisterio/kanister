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

package param

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"text/template"

	"github.com/pkg/errors"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/ksprig"
)

const (
	undefinedKeyErrorMsg = "map has no entry for key "
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
			rk, err := render(k.Interface(), tp)
			if err != nil {
				return nil, err
			}
			rv, err := render(val.MapIndex(k).Interface(), tp)
			if err != nil {
				return nil, err
			}
			ras[rk] = rv
		}
		return ras, nil
	case reflect.Struct:
		ras := reflect.New(val.Type())
		for i := 0; i < val.NumField(); i++ {
			r, err := render(val.Field(i).Interface(), tp)
			if err != nil {
				return nil, err
			}
			// set the field to the rendered value
			rv := reflect.Indirect(reflect.ValueOf(r))
			ras.Elem().Field(i).Set(rv)
		}
		return ras.Elem().Interface(), nil
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
		if a.KopiaSnapshot != "" {
			ks, err := renderStringArg(a.KopiaSnapshot, tp)
			if err != nil {
				return nil, err
			}
			ra.KopiaSnapshot = ks
		}
		rarts[name] = ra
	}
	return rarts, nil
}

func renderStringArg(arg string, tp TemplateParams) (string, error) {
	t, err := template.New("config").Option("missingkey=error").Funcs(ksprig.TxtFuncMap()).Parse(arg)
	if err != nil {
		return "", errors.WithStack(err)
	}
	buf := bytes.NewBuffer(nil)
	if err = t.Execute(buf, tp); err != nil {
		// Check if Error is because of undefined key,
		// which will lead on execute error on first undefined (missingkey=error).
		// Get the undefined key name from the error message.
		if strings.Contains(err.Error(), undefinedKeyErrorMsg) {
			return "", newUndefinedKeyError(err.Error())
		}
		return "", errors.WithStack(err)
	}
	return buf.String(), nil
}

func newUndefinedKeyError(err string) error {
	pos := strings.LastIndex(err, undefinedKeyErrorMsg)
	adjustedPos := pos + len(undefinedKeyErrorMsg)
	key := strings.Trim(err[adjustedPos:], "\"")
	return errors.WithStack(errors.New(fmt.Sprintf("Failed to render template: \"%s\" not found", key)))
}

// RenderObjectRefs function renders object refs from TemplateParams
func RenderObjectRefs(in map[string]crv1alpha1.ObjectReference, tp TemplateParams) (map[string]crv1alpha1.ObjectReference, error) {
	if tp.Time == "" {
		return nil, nil
	}

	out := make(map[string]crv1alpha1.ObjectReference, len(in))
	for k, v := range in {
		rv, err := render(v, tp)
		if err != nil {
			return nil, errors.Wrapf(err, "could not render object reference {%s}", k)
		}
		out[k] = rv.(crv1alpha1.ObjectReference)
	}
	return out, nil
}
