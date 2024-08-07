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
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/output"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/utils"
)

func init() {
	_ = kanister.Register(&kubeExecFunc{})
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

type kubeExecFunc struct {
	progressPercent string
}

func (*kubeExecFunc) Name() string {
	return KubeExecFuncName
}

func parseLogAndCreateOutput(out string) (map[string]interface{}, error) {
	if out == "" {
		return nil, nil
	}

	reader := io.NopCloser(strings.NewReader(out))
	output, err := output.LogAndParse(context.Background(), reader)

	// For some reason we expect empty output to be returned as nil here
	if len(output) == 0 {
		return nil, err
	}
	return output, err
}

func (kef *kubeExecFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	// Set progress percent
	kef.progressPercent = progress.StartedPercent
	defer func() { kef.progressPercent = progress.CompletedPercent }()

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

	var (
		bufStdout  = &bytes.Buffer{}
		outWriters = io.MultiWriter(os.Stdout, bufStdout)
	)
	if err := kube.ExecOutput(ctx, cli, namespace, pod, container, cmd, nil, outWriters, os.Stderr); err != nil {
		return nil, err
	}

	return parseLogAndCreateOutput(bufStdout.String())
}

func (*kubeExecFunc) RequiredArgs() []string {
	return []string{
		KubeExecNamespaceArg,
		KubeExecPodNameArg,
		KubeExecCommandArg,
	}
}

func (*kubeExecFunc) Arguments() []string {
	return []string{
		KubeExecNamespaceArg,
		KubeExecPodNameArg,
		KubeExecCommandArg,
		KubeExecContainerNameArg,
	}
}

func (kef *kubeExecFunc) Validate(args map[string]any) error {
	if err := utils.CheckSupportedArgs(kef.Arguments(), args); err != nil {
		return err
	}

	return utils.CheckRequiredArgs(kef.RequiredArgs(), args)
}

func (kef *kubeExecFunc) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    kef.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}
