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
	"strings"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/utils"
)

func init() {
	_ = kanister.Register(&kubeExecAllFunc{})
}

var (
	_ kanister.Func = (*kubeExecAllFunc)(nil)
)

const (
	// KubeExecAllFuncName gives the function name
	KubeExecAllFuncName          = "KubeExecAll"
	KubeExecAllNamespaceArg      = "namespace"
	KubeExecAllPodsNameArg       = "pods"
	KubeExecAllContainersNameArg = "containers"
	KubeExecAllCommandArg        = "command"
)

type kubeExecAllFunc struct {
	progressPercent string
}

func (*kubeExecAllFunc) Name() string {
	return KubeExecAllFuncName
}

func (kef *kubeExecAllFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	// Set progress percent
	kef.progressPercent = progress.StartedPercent
	defer func() { kef.progressPercent = progress.CompletedPercent }()

	cli, err := kube.NewClient()
	if err != nil {
		return nil, err
	}
	var namespace, pods, containers string
	var cmd []string
	if err = Arg(args, KubeExecAllNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, KubeExecAllPodsNameArg, &pods); err != nil {
		return nil, err
	}
	if err = Arg(args, KubeExecAllContainersNameArg, &containers); err != nil {
		return nil, err
	}
	if err = Arg(args, KubeExecAllCommandArg, &cmd); err != nil {
		return nil, err
	}
	ps := strings.Fields(pods)
	cs := strings.Fields(containers)
	return execAll(ctx, cli, namespace, ps, cs, cmd)
}

func (*kubeExecAllFunc) RequiredArgs() []string {
	return []string{
		KubeExecAllNamespaceArg,
		KubeExecAllPodsNameArg,
		KubeExecAllContainersNameArg,
		KubeExecAllCommandArg,
	}
}

func (*kubeExecAllFunc) Arguments() []string {
	return []string{
		KubeExecAllNamespaceArg,
		KubeExecAllPodsNameArg,
		KubeExecAllContainersNameArg,
		KubeExecAllCommandArg,
	}
}

func (k *kubeExecAllFunc) Validate(args map[string]any) error {
	if err := utils.CheckSupportedArgs(k.Arguments(), args); err != nil {
		return err
	}

	return utils.CheckRequiredArgs(k.RequiredArgs(), args)
}

func (k *kubeExecAllFunc) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    k.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}

func execAll(ctx context.Context, cli kubernetes.Interface, namespace string, ps []string, cs []string, cmd []string) (map[string]interface{}, error) {
	numContainers := len(ps) * len(cs)
	errChan := make(chan error, numContainers)
	output := ""
	// Run the command
	for _, p := range ps {
		for _, c := range cs {
			go func(p string, c string) {
				stdout, stderr, err := kube.Exec(ctx, cli, namespace, p, c, cmd, nil)
				format.LogWithCtx(ctx, p, c, stdout)
				format.LogWithCtx(ctx, p, c, stderr)
				errChan <- err
				output = output + "\n" + stdout
			}(p, c)
		}
	}
	errs := make([]string, 0, numContainers)
	for i := 0; i < numContainers; i++ {
		err := <-errChan
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) != 0 {
		return nil, errors.New(strings.Join(errs, "\n"))
	}
	out, err := parseLogAndCreateOutput(output)
	if err != nil {
		return nil, err
	}
	return out, nil
}
