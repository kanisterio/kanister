// Copyright 2023 The Kanister Authors.
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
	"strings"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/client-go/dynamic"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/utils"
)

const (
	// WaitV2FuncName specifies the function name
	WaitV2FuncName      = "WaitV2"
	WaitV2TimeoutArg    = "timeout"
	WaitV2ConditionsArg = "conditions"
)

func init() {
	_ = kanister.Register(&waitV2Func{})
}

var _ kanister.Func = (*waitV2Func)(nil)

type waitV2Func struct {
	progressPercent string
}

func (*waitV2Func) Name() string {
	return WaitV2FuncName
}

func (w *waitV2Func) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	// Set progress percent
	w.progressPercent = progress.StartedPercent
	defer func() { w.progressPercent = progress.CompletedPercent }()

	var timeout string
	if err := Arg(args, WaitV2TimeoutArg, &timeout); err != nil {
		return nil, err
	}

	// get the 'conditions' from the unrendered arguments list.
	// they will be evaluated in the 'evaluateCondition()` function.
	var conditions WaitConditions
	if err := Arg(args, WaitV2ConditionsArg, &conditions); err != nil {
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
	err = waitForCondition(ctx, dynCli, conditions, timeoutDur, tp, evaluateWaitV2Condition)
	return nil, err
}

func (*waitV2Func) RequiredArgs() []string {
	return []string{
		WaitV2TimeoutArg,
		WaitV2ConditionsArg,
	}
}

func (*waitV2Func) Arguments() []string {
	return []string{
		WaitV2TimeoutArg,
		WaitV2ConditionsArg,
	}
}

func (w *waitV2Func) Validate(args map[string]any) error {
	if err := utils.CheckSupportedArgs(w.Arguments(), args); err != nil {
		return err
	}

	return utils.CheckRequiredArgs(w.RequiredArgs(), args)
}

func (w *waitV2Func) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    w.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}

// evaluateWaitV2Condition evaluate the go template condition
func evaluateWaitV2Condition(ctx context.Context, dynCli dynamic.Interface, cond Condition, tp param.TemplateParams) (bool, error) {
	objRef, err := resolveWaitConditionObjRefs(cond, tp)
	if err != nil {
		return false, err
	}

	obj, err := fetchObjectFromRef(ctx, dynCli, objRef)
	if err != nil {
		return false, err
	}
	value, err := evaluateGoTemplate(obj, cond.Condition)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(value) == "true", nil
}

func evaluateGoTemplate(obj runtime.Object, goTemplateStr string) (string, error) {
	var buff bytes.Buffer
	jp, err := printers.NewGoTemplatePrinter([]byte(goTemplateStr))
	if err != nil {
		return "", nil
	}
	err = jp.PrintObj(obj, &buff)
	return buff.String(), err
}
