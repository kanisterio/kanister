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
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/utils"
)

func init() {
	_ = kanister.Register(&kubeops{})
}

var (
	_ kanister.Func = (*kubeops)(nil)
)

const (
	// KubeOpsFuncName gives the name of the function
	KubeOpsFuncName = "KubeOps"
	// KubeOpsSpecArg provides resource spec yaml
	KubeOpsSpecArg = "spec"
	// KubeOpsNamespaceArg provides resource namespace
	KubeOpsNamespaceArg = "namespace"
	// KubeOpsObjectReference specifies object details for delete operation
	KubeOpsObjectReferenceArg = "objectReference"
	// KubeOpsOperationArg is the kubeops operation needs to be executed
	KubeOpsOperationArg = "operation"
)

type kubeops struct {
	progressPercent string
}

func (*kubeops) Name() string {
	return KubeOpsFuncName
}

func (k *kubeops) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	// Set progress percent
	k.progressPercent = progress.StartedPercent
	defer func() { k.progressPercent = progress.CompletedPercent }()

	var spec, namespace string
	var op kube.Operation
	var objRefArg crv1alpha1.ObjectReference
	if err := OptArg(args, KubeOpsSpecArg, &spec, ""); err != nil {
		return nil, err
	}
	if err := Arg(args, KubeOpsOperationArg, &op); err != nil {
		return nil, err
	}
	if err := OptArg(args, KubeOpsNamespaceArg, &namespace, metav1.NamespaceDefault); err != nil {
		return nil, err
	}
	if ArgExists(args, KubeOpsObjectReferenceArg) {
		if err := OptArg(args, KubeOpsObjectReferenceArg, &objRefArg, nil); err != nil {
			return nil, err
		}
	}
	dynCli, err := kube.NewDynamicClient()
	if err != nil {
		return nil, err
	}
	objRef, err := execKubeOperation(ctx, dynCli, op, namespace, spec, objRefArg)
	if err != nil {
		return nil, err
	}
	objRefJSON, err := json.Marshal(objRef)
	if err != nil {
		return nil, err
	}
	// Convert objRef to map[string]interface{}
	var out map[string]interface{}
	if err := json.Unmarshal(objRefJSON, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func execKubeOperation(ctx context.Context, dynCli dynamic.Interface, op kube.Operation, namespace, spec string, objRef crv1alpha1.ObjectReference) (*crv1alpha1.ObjectReference, error) {
	kubeopsOp := kube.NewKubectlOperations(dynCli)
	switch op {
	case kube.CreateOperation:
		if len(spec) == 0 {
			return nil, errors.New(fmt.Sprintf("spec cannot be empty for %s operation", kube.CreateOperation))
		}
		return kubeopsOp.Create(strings.NewReader(spec), namespace)
	case kube.DeleteOperation:
		if objRef.Name == "" ||
			objRef.APIVersion == "" ||
			objRef.Resource == "" {
			return nil, errors.New(fmt.Sprintf("missing one or more required fields name/namespace/group/apiVersion/resource in objectReference for %s operation", kube.DeleteOperation))
		}
		return kubeopsOp.Delete(ctx, objRef, namespace)
	}
	return nil, errors.New(fmt.Sprintf("invalid operation '%s'", op))
}

func (*kubeops) RequiredArgs() []string {
	return []string{
		KubeOpsOperationArg,
	}
}

func (*kubeops) Arguments() []string {
	return []string{
		KubeOpsSpecArg,
		KubeOpsOperationArg,
		KubeOpsNamespaceArg,
		KubeOpsObjectReferenceArg,
	}
}

func (k *kubeops) Validate(args map[string]any) error {
	if err := utils.CheckSupportedArgs(k.Arguments(), args); err != nil {
		return err
	}

	return utils.CheckRequiredArgs(k.RequiredArgs(), args)
}

func (k *kubeops) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    k.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}
