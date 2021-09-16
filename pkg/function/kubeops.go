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

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	// KubeOpsSpecsArg provides resource spec yaml
	KubeOpsSpecsArg = "spec"
	// KubeOpsNamespaceArg provides resource namespace
	KubeOpsNamespaceArg = "namespace"
	// KubeOpsOperationArg is the kubeops operation needs to be executed
	KubeOpsOperationArg = "operation"
)

type kubeops struct{}

func (*kubeops) Name() string {
	return KubeOpsFuncName
}

func (crs *kubeops) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var spec, namespace string
	var op kube.Operation
	if err := Arg(args, KubeOpsSpecsArg, &spec); err != nil {
		return nil, err
	}
	if err := Arg(args, KubeOpsOperationArg, &op); err != nil {
		return nil, err
	}
	if err := OptArg(args, KubeOpsNamespaceArg, &namespace, metav1.NamespaceDefault); err != nil {
		return nil, err
	}
	kubeopsOp := kube.NewKubectlOperations(spec, namespace)
	objRef, err := kubeopsOp.Execute(op)
	if err != nil {
		return nil, err
	}
	objRefJson, err := json.Marshal(objRef)
	if err != nil {
		return nil, err
	}
	// Convert objRef to map[string]interface{}
	var out map[string]interface{}
	if err := json.Unmarshal(objRefJson, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (*kubeops) RequiredArgs() []string {
	return []string{
		KubeOpsSpecsArg,
		KubeOpsOperationArg,
	}
}
