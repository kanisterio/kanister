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

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
)

func init() {
	_ = kanister.Register(&kubectl{})
}

var (
	_ kanister.Func = (*kubectl)(nil)
)

const (
	// KubectlFuncName gives the name of the function
	KubectlFuncName = "kubectl"
	// KubectlSpecsArg provides resource specs yaml
	KubectlSpecsArg = "specs"
	// KubectlOperationArg is the kubectl operation needs to be executed
	KubectlOperationArg = "operation"
)

type kubectl struct{}

func (*kubectl) Name() string {
	return KubectlFuncName
}

func (crs *kubectl) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var specs string
	var op kube.Operation
	if err := Arg(args, KubectlSpecsArg, &specs); err != nil {
		return nil, err
	}
	if err := Arg(args, KubectlOperationArg, &op); err != nil {
		return nil, err
	}
	kubectlOp := kube.NewKubectlOperations(specs)
	return nil, kubectlOp.Execute(op)
}

func (*kubectl) RequiredArgs() []string {
	return []string{
		KubectlSpecsArg,
		KubectlOperationArg,
	}
}
