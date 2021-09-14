// Copyright 2021 The Kanister Authors.
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

package function

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/jsonpath"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/poll"
)

type WaitConditions struct {
	AnyOf []Condition
	AllOf []Condition
}

type Condition struct {
	ObjectReference crv1alpha1.ObjectReference
	Condition       string
}

const (
	WaitFuncName      = "Wait"
	WaitTimeoutArg    = "timeout"
	WaitConditionsArg = "conditions"
)

func init() {
	_ = kanister.Register(&waitFunc{})
}

var _ kanister.Func = (*waitFunc)(nil)

type waitFunc struct{}

func (*waitFunc) Name() string {
	return WaitFuncName
}

func (ktf *waitFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var timeout string
	var conditions WaitConditions
	var err error
	if err = Arg(args, WaitTimeoutArg, &timeout); err != nil {
		return nil, err
	}
	if err = Arg(args, WaitConditionsArg, &conditions); err != nil {
		return nil, err
	}
	dynCli, err := kube.NewDynamicClient()
	if err != nil {
		return nil, err
	}
	timeoutDur, err := time.ParseDuration(timeout)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse timeout")
	}
	err = waitForCondition(ctx, dynCli, conditions, timeoutDur)
	return nil, err
}

func (*waitFunc) RequiredArgs() []string {
	return []string{WaitTimeoutArg, WaitConditionsArg}
}

func waitForCondition(ctx context.Context, dynCli dynamic.Interface, waitCond WaitConditions, timeout time.Duration) error {
	ctxTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var evalErr error
	result := false
	err := poll.Wait(ctxTimeout, func(ctx context.Context) (bool, error) {
		for _, cond := range waitCond.AnyOf {
			result, evalErr = evaluateCondition(ctx, dynCli, cond)
			if evalErr != nil {
				// TODO: Fail early if the error is due to jsonpath syntax
				log.Debug().WithError(evalErr).Print("Failed to evaluate the condition")
				return false, nil
			}
			if result {
				return true, nil
			}

		}
		for _, cond := range waitCond.AllOf {
			result, evalErr = evaluateCondition(ctx, dynCli, cond)
			if evalErr != nil {
				// TODO: Fail early if the error is due to jsonpath syntax
				log.Debug().WithError(evalErr).Print("Failed to evaluate the condition")
				return false, nil
			}
			if !result {
				return false, nil
			}
		}
		return false, nil
	})
	if evalErr != nil {
		return errors.Wrap(err, evalErr.Error())
	}
	return err
}

func evaluateCondition(ctx context.Context, dynCli dynamic.Interface, cond Condition) (bool, error) {
	obj, err := fetchObjectFromRef(ctx, dynCli, cond.ObjectReference)
	if err != nil {
		return false, err
	}
	rcondition, err := resolveJsonpath(obj, cond.Condition)
	if err != nil {
		return false, err
	}
	log.Debug().Print(fmt.Sprintf("Resolved jsonpath: %s", rcondition))
	t, err := template.New("config").Option("missingkey=zero").Funcs(sprig.TxtFuncMap()).Parse(rcondition)
	if err != nil {
		return false, errors.WithStack(err)
	}
	buf := bytes.NewBuffer(nil)
	if err = t.Execute(buf, nil); err != nil {
		return false, errors.WithStack(err)
	}
	return strings.TrimSpace(buf.String()) == "true", nil
}

func fetchObjectFromRef(ctx context.Context, dynCli dynamic.Interface, objRef crv1alpha1.ObjectReference) (runtime.Object, error) {
	gvr := schema.GroupVersionResource{Group: objRef.Group, Version: objRef.APIVersion, Resource: objRef.Resource}
	if objRef.Namespace != "" {
		return dynCli.Resource(gvr).Namespace(objRef.Namespace).Get(ctx, objRef.Name, metav1.GetOptions{})
	}
	return dynCli.Resource(gvr).Get(ctx, objRef.Name, metav1.GetOptions{})
}

func resolveJsonpath(obj runtime.Object, condStr string) (string, error) {
	resolvedCondStr := condStr

	for s, match := range jsonpath.FindJsonpathArgs(condStr) {
		transCond := fmt.Sprintf("{%s}", strings.TrimSpace(match))
		value, err := jsonpath.ResolveJsonpathToString(obj, transCond)
		if err != nil {
			return "", err
		}
		resolvedCondStr = strings.ReplaceAll(resolvedCondStr, s, fmt.Sprintf("%s", value))
	}
	return resolvedCondStr, nil
}
