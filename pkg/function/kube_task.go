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
	"path"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/ephemeral"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/output"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/utils"
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

type kubeTaskFunc struct {
	progressPercent string
}

func (*kubeTaskFunc) Name() string {
	return KubeTaskFuncName
}

func kubeTask(
	ctx context.Context,
	cli kubernetes.Interface,
	namespace,
	image string,
	command []string,
	podOverride crv1alpha1.JSONMap,
	annotations,
	labels map[string]string,
) (map[string]interface{}, error) {
	options := &kube.PodOptions{
		Namespace:    namespace,
		GenerateName: jobPrefix,
		Image:        image,
		Command:      command,
		PodOverride:  podOverride,
		Annotations:  annotations,
		Labels:       labels,
	}

	// Apply the registered ephemeral pod changes.
	ephemeral.PodOptions.Apply(options)

	// Mark pod with label having key `kanister.io/JobID`, the value of which is a reference to the origin of the pod.
	kube.AddLabelsToPodOptionsFromContext(ctx, options, path.Join(consts.LabelPrefix, consts.LabelSuffixJobID))
	pr := kube.NewPodRunner(cli, options)
	podFunc := kubeTaskPodFunc()
	return pr.Run(ctx, podFunc)
}

func kubeTaskPodFunc() func(ctx context.Context, pc kube.PodController) (map[string]interface{}, error) {
	return func(ctx context.Context, pc kube.PodController) (map[string]interface{}, error) {
		if err := pc.WaitForPodReady(ctx); err != nil {
			return nil, errors.Wrapf(err, "Failed while waiting for Pod %s to be ready", pc.PodName())
		}
		ctx = field.Context(ctx, consts.LogKindKey, consts.LogKindDatapath)
		// Fetch logs from the pod
		r, err := pc.StreamPodLogs(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to fetch logs from the pod")
		}
		out, err := output.LogAndParse(ctx, r)
		if err != nil {
			return nil, err
		}
		// Wait for pod completion
		if err := pc.WaitForPodCompletion(ctx); err != nil {
			return nil, errors.Wrapf(err, "Failed while waiting for Pod %s to complete", pc.PodName())
		}
		return out, err
	}
}

func (ktf *kubeTaskFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	// Set progress percent
	ktf.progressPercent = progress.StartedPercent
	defer func() { ktf.progressPercent = progress.CompletedPercent }()

	var namespace, image string
	var command []string
	var err error
	var bpAnnotations, bpLabels map[string]string
	if err = Arg(args, KubeTaskImageArg, &image); err != nil {
		return nil, err
	}
	if err = Arg(args, KubeTaskCommandArg, &command); err != nil {
		return nil, err
	}
	if err = OptArg(args, KubeTaskNamespaceArg, &namespace, ""); err != nil {
		return nil, err
	}
	if err = OptArg(args, PodAnnotationsArg, &bpAnnotations, nil); err != nil {
		return nil, err
	}
	if err = OptArg(args, PodLabelsArg, &bpLabels, nil); err != nil {
		return nil, err
	}

	podOverride, err := GetPodSpecOverride(tp, args, KubeTaskPodOverrideArg)
	if err != nil {
		return nil, err
	}

	annotations := bpAnnotations
	labels := bpLabels
	if tp.PodAnnotations != nil {
		// merge the actionset annotations with blueprint annotations
		var actionSetAnn ActionSetAnnotations = tp.PodAnnotations
		annotations = actionSetAnn.MergeBPAnnotations(bpAnnotations)
	}

	if tp.PodLabels != nil {
		// merge the actionset labels with blueprint labels
		var actionSetLabels ActionSetLabels = tp.PodLabels
		labels = actionSetLabels.MergeBPLabels(bpLabels)
	}

	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	return kubeTask(
		ctx,
		cli,
		namespace,
		image,
		command,
		podOverride,
		annotations,
		labels,
	)
}

func (*kubeTaskFunc) RequiredArgs() []string {
	return []string{
		KubeTaskImageArg,
		KubeTaskCommandArg,
	}
}

func (*kubeTaskFunc) Arguments() []string {
	return []string{
		KubeTaskImageArg,
		KubeTaskCommandArg,
		KubeTaskNamespaceArg,
		KubeTaskPodOverrideArg,
		PodAnnotationsArg,
		PodLabelsArg,
	}
}

func (ktf *kubeTaskFunc) Validate(args map[string]any) error {
	if err := ValidatePodLabelsAndAnnotations(ktf.Name(), args); err != nil {
		return err
	}

	if err := utils.CheckSupportedArgs(ktf.Arguments(), args); err != nil {
		return err
	}

	return utils.CheckRequiredArgs(ktf.RequiredArgs(), args)
}

func (k *kubeTaskFunc) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    k.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}
