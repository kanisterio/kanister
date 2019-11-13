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
	"regexp"

	"github.com/pkg/errors"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/output"
	"github.com/kanisterio/kanister/pkg/param"
)

func init() {
	kanister.Register(&kubeExecFunc{})
}

var (
	_ kanister.Func = (*kubeExecFunc)(nil)
)

const (
	// KubeExecFuncName gives the function name
	KubeExecFuncName         = "KubeExec"
	KubeExecNamespaceArg     = "namespace"
	KubeExecPodNameArg       = "pod"
	KubeExecContainerNameArg = "container"
	KubeExecCommandArg       = "command"
)

type kubeExecFunc struct{}

func (*kubeExecFunc) Name() string {
	return KubeExecFuncName
}

func parseLogAndCreateOutput(out string) (map[string]interface{}, error) {
	if out == "" {
		return nil, nil
	}
	var op map[string]interface{}
	logs := regexp.MustCompile("[\n]").Split(out, -1)
	for _, l := range logs {
		opObj, err := output.Parse(l)
		if err != nil {
			return nil, err
		}
		if opObj == nil {
			continue
		}
		if op == nil {
			op = make(map[string]interface{})
		}
		op[opObj.Key] = opObj.Value
	}
	return op, nil
}

func (kef *kubeExecFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	cli, err := kube.NewClient()
	if err != nil {
		return nil, err
	}
	var namespace, pod, container string
	var cmd []string
	if err = Arg(args, KubeExecNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, KubeExecPodNameArg, &pod); err != nil {
		return nil, err
	}
	if err = OptArg(args, KubeExecContainerNameArg, &container, ""); err != nil {
		return nil, err
	}
	if err = Arg(args, KubeExecCommandArg, &cmd); err != nil {
		return nil, err
	}
	ctx = field.Context(ctx, consts.PodNameKey, pod)
	ctx = field.Context(ctx, consts.ContainerNameKey, container)
	stdout, stderr, err := kube.Exec(cli, namespace, pod, container, cmd, nil)
	format.Log(pod, container, stdout)
	format.Log(pod, container, stderr)
	if err != nil {
		return nil, err
	}

	out, err := parseLogAndCreateOutput(stdout)
	return out, errors.Wrap(err, "Failed to generate output")
}

func (*kubeExecFunc) RequiredArgs() []string {
	return []string{KubeExecNamespaceArg, KubeExecPodNameArg, KubeExecCommandArg}
}
