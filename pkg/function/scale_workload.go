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

package function

import (
	"context"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
)

const (
	// ScaleWorkloadFuncName gives the function name
	ScaleWorkloadFuncName     = "ScaleWorkload"
	ScaleWorkloadNamespaceArg = "namespace"
	ScaleWorkloadNameArg      = "name"
	ScaleWorkloadKindArg      = "kind"
	ScaleWorkloadReplicas     = "replicas"
)

func init() {
	kanister.Register(&scaleWorkloadFunc{})
}

var (
	_ kanister.Func = (*scaleWorkloadFunc)(nil)
)

type scaleWorkloadFunc struct{}

func (*scaleWorkloadFunc) Name() string {
	return ScaleWorkloadFuncName
}

func (*scaleWorkloadFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var namespace, kind, name string
	var replicas int32
	namespace, kind, name, replicas, err := getArgs(tp, args)
	if err != nil {
		return nil, err
	}

	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	switch strings.ToLower(kind) {
	case param.StatefulSetKind:
		return nil, kube.ScaleStatefulSet(ctx, cli, namespace, name, replicas)
	case param.DeploymentKind:
		return nil, kube.ScaleDeployment(ctx, cli, namespace, name, replicas)
	default:
		return nil, errors.New("Workload type not supported " + kind)
	}
}

func (*scaleWorkloadFunc) RequiredArgs() []string {
	return []string{ScaleWorkloadReplicas}
}

func getArgs(tp param.TemplateParams, args map[string]interface{}) (namespace, kind, name string, replicas int32, err error) {
	var rep interface{}
	err = Arg(args, ScaleWorkloadReplicas, &rep)
	if err != nil {
		return namespace, kind, name, replicas, err
	}

	switch val := rep.(type) {
	case int:
		replicas = int32(val)
	case int32:
		replicas = val
	case int64:
		replicas = int32(val)
	case string:
		var v int
		if v, err = strconv.Atoi(val); err != nil {
			err = errors.Wrapf(err, "Cannot convert %s to int ", val)
			return
		}
		replicas = int32(v)
	default:
		err = errors.Errorf("Invalid arg type %T for Arg %s ", rep, ScaleWorkloadReplicas)
		return
	}
	// Populate default values for optional arguments from template parameters
	switch {
	case tp.StatefulSet != nil:
		kind = param.StatefulSetKind
		name = tp.StatefulSet.Name
		namespace = tp.StatefulSet.Namespace
	case tp.Deployment != nil:
		kind = param.DeploymentKind
		name = tp.Deployment.Name
		namespace = tp.Deployment.Namespace
	default:
		if !ArgExists(args, ScaleWorkloadNamespaceArg) || !ArgExists(args, ScaleWorkloadNameArg) || !ArgExists(args, ScaleWorkloadKindArg) {
			return namespace, kind, name, replicas, errors.New("Workload information not available via defaults or namespace/name/kind parameters")
		}
	}

	err = OptArg(args, ScaleWorkloadNamespaceArg, &namespace, namespace)
	if err != nil {
		return namespace, kind, name, replicas, err
	}
	err = OptArg(args, ScaleWorkloadNameArg, &name, name)
	if err != nil {
		return namespace, kind, name, replicas, err
	}
	err = OptArg(args, ScaleWorkloadKindArg, &kind, kind)
	if err != nil {
		return namespace, kind, name, replicas, err
	}
	return namespace, kind, name, replicas, err
}
