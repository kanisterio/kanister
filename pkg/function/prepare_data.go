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
	"fmt"
	"io"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/ephemeral"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/utils"
)

const (
	defaultMountPoint    = "/mnt/prepare_data/%s"
	prepareDataJobPrefix = "prepare-data-job-"
	// PrepareDataFuncName gives the function name
	PrepareDataFuncName       = "PrepareData"
	PrepareDataNamespaceArg   = "namespace"
	PrepareDataImageArg       = "image"
	PrepareDataCommandArg     = "command"
	PrepareDataVolumes        = "volumes"
	PrepareDataServiceAccount = "serviceaccount"
	PrepareDataPodOverrideArg = "podOverride"
)

func init() {
	_ = kanister.Register(&prepareDataFunc{})
}

var _ kanister.Func = (*prepareDataFunc)(nil)

type prepareDataFunc struct {
	progressPercent string
}

func (*prepareDataFunc) Name() string {
	return PrepareDataFuncName
}

func getVolumes(tp param.TemplateParams) (map[string]string, error) {
	vols := make(map[string]string)
	var podsToPvcs map[string]map[string]string
	switch {
	case tp.Deployment != nil:
		podsToPvcs = tp.Deployment.PersistentVolumeClaims
	case tp.StatefulSet != nil:
		podsToPvcs = tp.StatefulSet.PersistentVolumeClaims
	default:
		return nil, errors.New("Failed to get volumes")
	}
	for _, podToPvcs := range podsToPvcs {
		for pvc := range podToPvcs {
			vols[pvc] = fmt.Sprintf(defaultMountPoint, pvc)
		}
	}
	if len(vols) == 0 {
		return nil, errors.New("No volumes found")
	}
	return vols, nil
}

func prepareData(
	ctx context.Context,
	cli kubernetes.Interface,
	namespace,
	serviceAccount,
	image string,
	vols map[string]string,
	podOverride crv1alpha1.JSONMap,
	annotations,
	labels map[string]string,
	command ...string,
) (map[string]interface{}, error) {
	// Validate volumes
	validatedVols := make(map[string]kube.VolumeMountOptions)
	for pvcName, mountPoint := range vols {
		pvc, err := cli.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, pvcName, metav1.GetOptions{})
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to retrieve PVC. Namespace %s, Name %s", namespace, pvcName)
		}

		validatedVols[pvcName] = kube.VolumeMountOptions{
			MountPath: mountPoint,
			ReadOnly:  kube.PVCContainsReadOnlyAccessMode(pvc),
		}
	}

	options := &kube.PodOptions{
		Namespace:          namespace,
		GenerateName:       prepareDataJobPrefix,
		Image:              image,
		Command:            command,
		Volumes:            validatedVols,
		ServiceAccountName: serviceAccount,
		PodOverride:        podOverride,
		Annotations:        annotations,
		Labels:             labels,
	}

	// Apply the registered ephemeral pod changes.
	ephemeral.PodOptions.Apply(options)

	pr := kube.NewPodRunner(cli, options)
	podFunc := prepareDataPodFunc(cli)
	return pr.Run(ctx, podFunc)
}

func prepareDataPodFunc(cli kubernetes.Interface) func(ctx context.Context, pc kube.PodController) (map[string]interface{}, error) {
	return func(ctx context.Context, pc kube.PodController) (map[string]interface{}, error) {
		pod := pc.Pod()

		// Wait for pod to reach running state
		if err := pc.WaitForPodReady(ctx); err != nil {
			return nil, errors.Wrapf(err, "Failed while waiting for Pod %s to be ready", pod.Name)
		}

		ctx = field.Context(ctx, consts.LogKindKey, consts.LogKindDatapath)
		// Fetch logs from the pod
		r, err := pc.StreamPodLogs(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to fetch logs from the pod")
		}
		defer r.Close() //nolint:errcheck

		bytes, err := io.ReadAll(r)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to read logs from the pod")
		}
		logs := string(bytes)

		format.LogWithCtx(ctx, pod.Name, pod.Spec.Containers[0].Name, logs)
		out, err := parseLogAndCreateOutput(logs)
		return out, errors.Wrap(err, "Failed to parse phase output")
	}
}

func (p *prepareDataFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	// Set progress percent
	p.progressPercent = progress.StartedPercent
	defer func() { p.progressPercent = progress.CompletedPercent }()

	var namespace, image, serviceAccount string
	var command []string
	var vols map[string]string
	var bpAnnotations, bpLabels map[string]string
	var err error
	if err = Arg(args, PrepareDataNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, PrepareDataImageArg, &image); err != nil {
		return nil, err
	}
	if err = Arg(args, PrepareDataCommandArg, &command); err != nil {
		return nil, err
	}
	if err = OptArg(args, PrepareDataVolumes, &vols, nil); err != nil {
		return nil, err
	}
	if err = OptArg(args, PrepareDataServiceAccount, &serviceAccount, ""); err != nil {
		return nil, err
	}
	if err = OptArg(args, PodAnnotationsArg, &bpAnnotations, nil); err != nil {
		return nil, err
	}
	if err = OptArg(args, PodLabelsArg, &bpLabels, nil); err != nil {
		return nil, err
	}
	podOverride, err := GetPodSpecOverride(tp, args, PrepareDataPodOverrideArg)
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
	if len(vols) == 0 {
		if vols, err = getVolumes(tp); err != nil {
			return nil, err
		}
	}
	return prepareData(
		ctx,
		cli,
		namespace,
		serviceAccount,
		image,
		vols,
		podOverride,
		annotations,
		labels,
		command...,
	)
}

func (*prepareDataFunc) RequiredArgs() []string {
	return []string{
		PrepareDataNamespaceArg,
		PrepareDataImageArg,
		PrepareDataCommandArg,
	}
}

func (*prepareDataFunc) Arguments() []string {
	return []string{
		PrepareDataNamespaceArg,
		PrepareDataImageArg,
		PrepareDataCommandArg,
		PrepareDataVolumes,
		PrepareDataServiceAccount,
		PrepareDataPodOverrideArg,
		PodAnnotationsArg,
		PodLabelsArg,
	}
}

func (p *prepareDataFunc) Validate(args map[string]any) error {
	if err := ValidatePodLabelsAndAnnotations(p.Name(), args); err != nil {
		return err
	}

	if err := utils.CheckSupportedArgs(p.Arguments(), args); err != nil {
		return err
	}

	return utils.CheckRequiredArgs(p.RequiredArgs(), args)
}

func (p *prepareDataFunc) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    p.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}
