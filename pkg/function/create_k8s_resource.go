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
	_ = kanister.Register(&createK8sResource{})
}

var (
	_ kanister.Func = (*createK8sResource)(nil)
)

const (
	// CreateK8sResourceFuncName gives the name of the function
	CreateK8sResourceFuncName = "CreateK8sResource"
	SpecsArg                  = "specs"
)

type createK8sResource struct{}

func (*createK8sResource) Name() string {
	return CreateK8sResourceFuncName
}

func (crs *createK8sResource) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var specs string
	if err := Arg(args, SpecsArg, &specs); err != nil {
		return nil, err
	}
	return nil, kube.CreateResourceFromSpecs(specs)
}

func (*createK8sResource) RequiredArgs() []string {
	return []string{
		SpecsArg,
	}
}
