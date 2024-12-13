//
// Copyright 2019 The Kanister Authors.
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
	"sort"
	"time"

	"github.com/kanisterio/errkit"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/output"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/utils"
)

const (
	// MultiContainerRunFuncName gives the function name
	MultiContainerRunFuncName             = "MultiContainerRun"
	MultiContainerRunNamespaceArg         = "namespace"
	MultiContainerRunBackgroundImageArg   = "backgroundImage"
	MultiContainerRunBackgroundCommandArg = "backgroundCommand"
	MultiContainerRunOutputImageArg       = "outputImage"
	MultiContainerRunOutputCommandArg     = "outputCommand"
	MultiContainerRunVolumeMediumArg      = "sharedVolumeMedium"
	MultiContainerRunVolumeSizeLimitArg   = "sharedVolumeSizeLimit"
	MultiContainerRunSharedDirArg         = "sharedVolumeDir"
	MultiContainerRunPodOverrideArg       = "podOverride"
	MultiContainerRunInitImageArg         = "initImage"
	MultiContainerRunInitCommandArg       = "initCommand"
)

const (
	ktpBackgroundContainer = "background"
	ktpOutputContainer     = "output"
	ktpSharedVolumeName    = "shared"
	ktpDefaultSharedDir    = "/tmp/"
)

func init() {
	_ = kanister.Register(&multiContainerRunFunc{})
}

var _ kanister.Func = (*multiContainerRunFunc)(nil)

type multiContainerRunFunc struct {
	progressPercent   string
	namespace         string
	backgroundImage   string
	backgroundCommand []string
	outputImage       string
	outputCommand     []string
	initImage         string
	initCommand       []string
	storageDir        string
	storageMedium     corev1.StorageMedium
	storageSizeLimit  *resource.Quantity
	podOverride       crv1alpha1.JSONMap
	labels            map[string]string
	annotations       map[string]string
}

func (*multiContainerRunFunc) Name() string {
	return MultiContainerRunFuncName
}

func (ktpf *multiContainerRunFunc) run(ctx context.Context) (map[string]interface{}, error) {
	cli, err := kube.NewClient()
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to create Kubernetes client")
	}

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      ktpSharedVolumeName,
			MountPath: ktpf.storageDir,
		},
	}

	var initContainers []corev1.Container
	// If init image is specified
	if ktpf.initImage != "" {
		initContainers = []corev1.Container{
			{
				Name:         "init",
				Image:        ktpf.initImage,
				Command:      ktpf.initCommand,
				VolumeMounts: volumeMounts,
			},
		}
	}

	podSpec := corev1.PodSpec{
		RestartPolicy: corev1.RestartPolicyNever,
		Volumes: []corev1.Volume{
			{
				Name: ktpSharedVolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{
						Medium:    ktpf.storageMedium,
						SizeLimit: ktpf.storageSizeLimit,
					},
				},
			},
		},
		InitContainers: initContainers,
		Containers: []corev1.Container{
			{
				Name:         ktpOutputContainer,
				Image:        ktpf.outputImage,
				Command:      ktpf.outputCommand,
				VolumeMounts: volumeMounts,
			},
			{
				Name:         ktpBackgroundContainer,
				Image:        ktpf.backgroundImage,
				Command:      ktpf.backgroundCommand,
				VolumeMounts: volumeMounts,
			},
		},
	}

	podSpec, err = kube.PatchDefaultPodSpecs(podSpec, ktpf.podOverride)
	if err != nil {
		return nil, errkit.Wrap(err, "Unable to apply podOverride", "podSpec", podSpec, "podOverride", ktpf.podOverride)
	}

	// Put the output container the first
	sort.Slice(podSpec.Containers, func(i, j int) bool {
		return podSpec.Containers[i].Name == ktpOutputContainer
	})

	if ktpf.labels == nil {
		ktpf.labels = make(map[string]string)
	}
	ktpf.labels[consts.LabelKeyCreatedBy] = consts.LabelValueKanister

	if ktpf.annotations == nil {
		ktpf.annotations = make(map[string]string)
	}
	// FIXME: this doesn't work with pod controller currently so we have to reorder containers
	ktpf.annotations[defaultContainerAnn] = ktpOutputContainer

	err = setPodSpecServiceAccount(&podSpec, ktpf.namespace, cli)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to set serviceaccount for pod")
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: jobPrefix,
			Namespace:    ktpf.namespace,
			Labels:       ktpf.labels,
			Annotations:  ktpf.annotations,
		},
		Spec: podSpec,
	}

	pod, err = cli.CoreV1().Pods(ktpf.namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to create pod")
	}
	pc, err := kube.NewPodControllerForExistingPod(cli, pod)
	if err != nil {
		return nil, err
	}

	ctx = field.Context(ctx, consts.PodNameKey, pod.Name)
	ctx = field.Context(ctx, consts.ContainerNameKey, pod.Spec.Containers[0].Name)
	go func() {
		<-ctx.Done()
		err := pc.StopPod(context.Background(), kube.PodControllerInfiniteStopTime, int64(0))
		if err != nil {
			log.WithError(err).Print("Failed to delete pod", field.M{"PodName": pod.Name})
		}
	}()

	return getPodOutput(ctx, pc)
}

func setPodSpecServiceAccount(podSpec *corev1.PodSpec, ns string, cli kubernetes.Interface) error {
	sa := podSpec.ServiceAccountName
	controllerNamespace, err := kube.GetControllerNamespace()
	if err != nil {
		return errkit.Wrap(err, "Failed to get controller namespace")
	}

	if sa == "" && ns == controllerNamespace {
		sa, err = kube.GetControllerServiceAccount(cli)
		if err != nil {
			return errkit.Wrap(err, "Failed to get Controller Service Account")
		}
	}
	podSpec.ServiceAccountName = sa
	return nil
}

// This function is similar to kubeTaskPodFunc
func getPodOutput(ctx context.Context, pc kube.PodController) (map[string]interface{}, error) {
	if err := pc.WaitForPodReady(ctx); err != nil {
		return nil, errkit.Wrap(err, "Failed while waiting for Pod to be ready", "pod", pc.PodName())
	}
	ctx = field.Context(ctx, consts.LogKindKey, consts.LogKindDatapath)
	// Fetch logs from the pod
	r, err := pc.StreamPodLogs(ctx)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to fetch logs from the pod")
	}
	out, err := output.LogAndParse(ctx, r)
	if err != nil {
		return nil, err
	}
	// Wait for pod completion
	if err := pc.WaitForPodCompletion(ctx); err != nil {
		return nil, errkit.Wrap(err, "Failed while waiting for Pod to complete", "pod", pc.PodName())
	}
	return out, err
}

func (ktpf *multiContainerRunFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	// Set progress percent
	ktpf.progressPercent = progress.StartedPercent
	defer func() { ktpf.progressPercent = progress.CompletedPercent }()

	var storageSizeString string
	var bpAnnotations, bpLabels map[string]string
	var err error
	if err = Arg(args, MultiContainerRunBackgroundImageArg, &ktpf.backgroundImage); err != nil {
		return nil, err
	}
	if err = Arg(args, MultiContainerRunOutputImageArg, &ktpf.outputImage); err != nil {
		return nil, err
	}
	if err = OptArg(args, MultiContainerRunInitImageArg, &ktpf.initImage, ""); err != nil {
		return nil, err
	}
	if err = Arg(args, MultiContainerRunBackgroundCommandArg, &ktpf.backgroundCommand); err != nil {
		return nil, err
	}
	if err = Arg(args, MultiContainerRunOutputCommandArg, &ktpf.outputCommand); err != nil {
		return nil, err
	}
	if err = OptArg(args, MultiContainerRunInitCommandArg, &ktpf.initCommand, nil); err != nil {
		return nil, err
	}

	if err = OptArg(args, MultiContainerRunNamespaceArg, &ktpf.namespace, ""); err != nil {
		return nil, err
	}

	if ktpf.namespace == "" {
		controllerNamespace, err := kube.GetControllerNamespace()
		if err != nil {
			return nil, errkit.Wrap(err, "Failed to get controller namespace")
		}
		ktpf.namespace = controllerNamespace
	}

	if err = OptArg(args, MultiContainerRunVolumeMediumArg, &ktpf.storageMedium, ""); err != nil {
		return nil, err
	}
	if err = OptArg(args, MultiContainerRunVolumeSizeLimitArg, &storageSizeString, ""); err != nil {
		return nil, err
	}
	if storageSizeString != "" {
		size, err := resource.ParseQuantity(storageSizeString)
		if err != nil {
			return nil, errkit.Wrap(err, "Failed to parse sharedStorageSize arg")
		}
		ktpf.storageSizeLimit = &size
	}
	if err = OptArg(args, MultiContainerRunSharedDirArg, &ktpf.storageDir, ktpDefaultSharedDir); err != nil {
		return nil, err
	}
	if err = OptArg(args, PodAnnotationsArg, &bpAnnotations, nil); err != nil {
		return nil, err
	}
	if err = OptArg(args, PodLabelsArg, &bpLabels, nil); err != nil {
		return nil, err
	}

	ktpf.podOverride, err = GetPodSpecOverride(tp, args, MultiContainerRunPodOverrideArg)
	if err != nil {
		return nil, err
	}

	ktpf.setLabelsAndAnnotations(tp, bpLabels, bpAnnotations)

	return ktpf.run(ctx)
}

func (ktpf *multiContainerRunFunc) setLabelsAndAnnotations(tp param.TemplateParams, labels, annotation map[string]string) {
	ktpf.labels = labels
	ktpf.annotations = annotation
	if tp.PodAnnotations != nil {
		// merge the actionset annotations with blueprint annotations
		var actionSetAnn ActionSetAnnotations = tp.PodAnnotations
		ktpf.annotations = actionSetAnn.MergeBPAnnotations(annotation)
	}

	if tp.PodLabels != nil {
		// merge the actionset labels with blueprint labels
		var actionSetLabels ActionSetLabels = tp.PodLabels
		ktpf.labels = actionSetLabels.MergeBPLabels(labels)
	}
}

func (*multiContainerRunFunc) RequiredArgs() []string {
	return []string{
		MultiContainerRunBackgroundImageArg,
		MultiContainerRunBackgroundCommandArg,
		MultiContainerRunOutputImageArg,
		MultiContainerRunOutputCommandArg,
	}
}

func (*multiContainerRunFunc) Arguments() []string {
	return []string{
		MultiContainerRunNamespaceArg,
		MultiContainerRunInitImageArg,
		MultiContainerRunInitCommandArg,
		MultiContainerRunBackgroundImageArg,
		MultiContainerRunBackgroundCommandArg,
		MultiContainerRunOutputImageArg,
		MultiContainerRunOutputCommandArg,
		MultiContainerRunVolumeMediumArg,
		MultiContainerRunVolumeSizeLimitArg,
		MultiContainerRunSharedDirArg,
		MultiContainerRunPodOverrideArg,
		PodLabelsArg,
		PodAnnotationsArg,
	}
}

func (ktpf *multiContainerRunFunc) Validate(args map[string]any) error {
	if err := ValidatePodLabelsAndAnnotations(ktpf.Name(), args); err != nil {
		return err
	}

	if err := utils.CheckSupportedArgs(ktpf.Arguments(), args); err != nil {
		return err
	}

	return utils.CheckRequiredArgs(ktpf.RequiredArgs(), args)
}

func (k *multiContainerRunFunc) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    k.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}
