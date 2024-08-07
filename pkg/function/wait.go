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

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/jsonpath"
	"github.com/kanisterio/kanister/pkg/ksprig"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/poll"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/utils"
)

type WaitConditions struct {
	AnyOf []Condition `json:"anyOf,omitempty"`
	AllOf []Condition `json:"allOf,omitempty"`
}

type Condition struct {
	ObjectReference crv1alpha1.ObjectReference `json:"objectReference,omitempty"`
	Condition       string                     `json:"condition,omitempty"`
}

const (
	// WaitFuncName specifies the function name
	WaitFuncName      = "Wait"
	WaitTimeoutArg    = "timeout"
	WaitConditionsArg = "conditions"
)

func init() {
	_ = kanister.Register(&waitFunc{})
}

var _ kanister.Func = (*waitFunc)(nil)

type waitFunc struct {
	progressPercent string
}

func (*waitFunc) Name() string {
	return WaitFuncName
}

func (w *waitFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	// Set progress percent
	w.progressPercent = progress.StartedPercent
	defer func() { w.progressPercent = progress.CompletedPercent }()

	rendered, err := param.RenderArgs(args, tp)
	if err != nil {
		return nil, err
	}

	var timeout string
	if err := Arg(rendered, WaitTimeoutArg, &timeout); err != nil {
		return nil, err
	}

	// get the 'conditions' from the unrendered arguments list.
	// they will be evaluated in the 'evaluateCondition()` function.
	var conditions WaitConditions
	if err := Arg(args, WaitConditionsArg, &conditions); err != nil {
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
	err = waitForCondition(ctx, dynCli, conditions, timeoutDur, tp, evaluateWaitCondition)
	return nil, err
}

func (*waitFunc) RequiredArgs() []string {
	return []string{
		WaitTimeoutArg,
		WaitConditionsArg,
	}
}

func (*waitFunc) Arguments() []string {
	return []string{
		WaitTimeoutArg,
		WaitConditionsArg,
	}
}

func (w *waitFunc) Validate(args map[string]any) error {
	if err := utils.CheckSupportedArgs(w.Arguments(), args); err != nil {
		return err
	}

	return utils.CheckRequiredArgs(w.RequiredArgs(), args)
}

func (w *waitFunc) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    w.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}

// waitForCondition wait till the condition satisfies within the timeout duration
func waitForCondition(
	ctx context.Context,
	dynCli dynamic.Interface,
	waitCond WaitConditions,
	timeout time.Duration,
	tp param.TemplateParams,
	eval func(context.Context, dynamic.Interface, Condition, param.TemplateParams) (bool, error),
) error {
	ctxTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var evalErr error
	result := false
	err := poll.Wait(ctxTimeout, func(ctx context.Context) (bool, error) {
		for _, cond := range waitCond.AnyOf {
			result, evalErr = eval(ctx, dynCli, cond, tp)
			if evalErr != nil {
				// TODO: Fail early if the error is due to invalid syntax
				log.Debug().WithError(evalErr).Print("Failed to evaluate the condition", field.M{"result": result})
				return false, nil
			}
			if result {
				return true, nil
			}
		}
		for _, cond := range waitCond.AllOf {
			result, evalErr = eval(ctx, dynCli, cond, tp)
			if evalErr != nil {
				// TODO: Fail early if the error is due to invalid syntax
				log.Debug().WithError(evalErr).Print("Failed to evaluate the condition", field.M{"result": result})
				return false, nil
			}
			// Retry if any condition fails
			if !result {
				return false, nil
			}
		}
		return result, nil
	})
	err = errors.Wrap(err, "Failed to wait for the condition to be met")
	if evalErr != nil {
		return errors.Wrap(err, evalErr.Error())
	}
	return err
}

// evaluateWaitCondition evaluate the go template condition
func evaluateWaitCondition(ctx context.Context, dynCli dynamic.Interface, cond Condition, tp param.TemplateParams) (bool, error) {
	objRef, err := resolveWaitConditionObjRefs(cond, tp)
	if err != nil {
		return false, err
	}

	obj, err := fetchObjectFromRef(ctx, dynCli, objRef)
	if err != nil {
		return false, err
	}
	rcondition, err := resolveJsonpath(obj, cond.Condition)
	if err != nil {
		return false, err
	}
	log.Debug().Print(fmt.Sprintf("Resolved jsonpath: %s", rcondition))
	t, err := template.New("config").Option("missingkey=zero").Funcs(ksprig.TxtFuncMap()).Parse(rcondition)
	if err != nil {
		return false, errors.WithStack(err)
	}
	buf := bytes.NewBuffer(nil)
	if err = t.Execute(buf, nil); err != nil {
		return false, errors.WithStack(err)
	}
	log.Debug().Print(fmt.Sprintf("Condition evaluation result: %s", strings.TrimSpace(buf.String())))
	return strings.TrimSpace(buf.String()) == "true", nil
}

func resolveWaitConditionObjRefs(cond Condition, tp param.TemplateParams) (crv1alpha1.ObjectReference, error) {
	objRefRaw := map[string]crv1alpha1.ObjectReference{
		"objRef": cond.ObjectReference,
	}
	rendered, err := param.RenderObjectRefs(objRefRaw, tp)
	if err != nil {
		return crv1alpha1.ObjectReference{}, err
	}
	return rendered["objRef"], nil
}

func fetchObjectFromRef(ctx context.Context, dynCli dynamic.Interface, objRef crv1alpha1.ObjectReference) (runtime.Object, error) {
	gvr := schema.GroupVersionResource{Group: objRef.Group, Version: objRef.APIVersion, Resource: objRef.Resource}
	if objRef.Namespace != "" {
		return dynCli.Resource(gvr).Namespace(objRef.Namespace).Get(ctx, objRef.Name, metav1.GetOptions{})
	}
	return dynCli.Resource(gvr).Get(ctx, objRef.Name, metav1.GetOptions{})
}

// resolveJsonpath resolves jsonpath fields and replaces the occurrences with actual values
func resolveJsonpath(obj runtime.Object, condStr string) (string, error) {
	resolvedCondStr := condStr
	for s, match := range jsonpath.FindJsonpathArgs(condStr) {
		transCond := fmt.Sprintf("{%s}", strings.TrimSpace(match))
		value, err := jsonpath.ResolveJsonpathToString(obj, transCond)
		if err != nil {
			return "", err
		}
		resolvedCondStr = strings.ReplaceAll(resolvedCondStr, s, value)
	}
	return resolvedCondStr, nil
}
