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

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/output"
	"github.com/kanisterio/kanister/pkg/param"
)

const (
	jobPrefix = "kanister-job-"
	// KubeTaskFuncName gives the function name
	KubeTaskFuncName       = "KubeTask"
	KubeTaskNamespaceArg   = "namespace"
	KubeTaskImageArg       = "image"
	KubeTaskCommandArg     = "command"
	KubeTaskPodOverrideArg = "podOverride"
)

func init() {
	_ = kanister.Register(&kubeTaskFunc{})
}

var _ kanister.Func = (*kubeTaskFunc)(nil)

type kubeTaskFunc struct{}

func (*kubeTaskFunc) Name() string {
	return KubeTaskFuncName
}

func kubeTask(ctx context.Context, cli kubernetes.Interface, namespace, image string, command []string, podOverride crv1alpha1.JSONMap) (map[string]interface{}, error) {
	var serviceAccount string
	var err error
	if namespace == "" {
		namespace, err = kube.GetControllerNamespace()
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to get controller namespace")
		}
		serviceAccount, err = kube.GetControllerServiceAccount(cli)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to get Controller Service Account")
		}
	}
	options := &kube.PodOptions{
		Namespace:          namespace,
		GenerateName:       jobPrefix,
		Image:              image,
		Command:            command,
		ServiceAccountName: serviceAccount,
		PodOverride:        podOverride,
	}

	pr := kube.NewPodRunner(cli, options)
	podFunc := kubeTaskPodFunc(cli)
	return pr.Run(ctx, podFunc)
}

func kubeTaskPodFunc(cli kubernetes.Interface) func(ctx context.Context, pod *v1.Pod) (map[string]interface{}, error) {
	return func(ctx context.Context, pod *v1.Pod) (map[string]interface{}, error) {
		if err := kube.WaitForPodReady(ctx, cli, pod.Namespace, pod.Name); err != nil {
			return nil, errors.Wrapf(err, "Failed while waiting for Pod %s to complete", pod.Name)
		}
		ctx = field.Context(ctx, consts.PodNameKey, pod.Name)
		// Fetch logs from the pod
		r, err := kube.StreamPodLogs(ctx, cli, pod.Namespace, pod.Name)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to fetch logs from the pod")
		}
		out, err := output.LogAndParse(ctx, r)
		if err != nil {
			return nil, err
		}
		// Wait for pod completion
		if err := kube.WaitForPodCompletion(ctx, cli, pod.Namespace, pod.Name); err != nil {
			return nil, errors.Wrapf(err, "Failed while waiting for Pod %s to complete", pod.Name)
		}
		return out, err
	}
}

func (ktf *kubeTaskFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var namespace, image string
	var command []string
	var err error
	if err = Arg(args, KubeTaskImageArg, &image); err != nil {
		return nil, err
	}
	if err = Arg(args, KubeTaskCommandArg, &command); err != nil {
		return nil, err
	}
	if err = OptArg(args, KubeTaskNamespaceArg, &namespace, ""); err != nil {
		return nil, err
	}
	podOverride, err := GetPodSpecOverride(tp, args, KubeTaskPodOverrideArg)
	if err != nil {
		return nil, err
	}

	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	return kubeTask(ctx, cli, namespace, image, command, podOverride)
}

func (*kubeTaskFunc) RequiredArgs() []string {
	return []string{KubeTaskNamespaceArg, KubeTaskImageArg, KubeTaskCommandArg}
}
