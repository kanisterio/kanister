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
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/ephemeral"
	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/restic"
	"github.com/kanisterio/kanister/pkg/utils"
)

const (
	restoreDataJobPrefix = "restore-data-"
	// RestoreDataFuncName gives the function name
	RestoreDataFuncName = "RestoreData"
	// RestoreDataNamespaceArg provides the namespace
	RestoreDataNamespaceArg = "namespace"
	// RestoreDataImageArg provides the image of the container with required tools
	RestoreDataImageArg = "image"
	// RestoreDataBackupArtifactPrefixArg provides the path of the backed up artifact
	RestoreDataBackupArtifactPrefixArg = "backupArtifactPrefix"
	// RestoreDataRestorePathArg provides the path to restore backed up data
	RestoreDataRestorePathArg = "restorePath"
	// RestoreDataBackupIdentifierArg provides a unique ID added to the backed up artifacts
	RestoreDataBackupIdentifierArg = "backupIdentifier"
	// RestoreDataPodArg provides the pod connected to the data volume
	RestoreDataPodArg = "pod"
	// RestoreDataVolsArg provides a map of PVC->mountPaths to be attached
	RestoreDataVolsArg = "volumes"
	// RestoreDataEncryptionKeyArg provides the encryption key used during backup
	RestoreDataEncryptionKeyArg = "encryptionKey"
	// RestoreDataBackupTagArg provides a unique tag added to the backup artifacts
	RestoreDataBackupTagArg = "backupTag"
	// RestoreDataPodOverrideArg contains pod specs which overrides default pod specs
	RestoreDataPodOverrideArg = "podOverride"
)

func init() {
	_ = kanister.Register(&restoreDataFunc{})
}

var _ kanister.Func = (*restoreDataFunc)(nil)

type restoreDataFunc struct {
	progressPercent string
}

func (*restoreDataFunc) Name() string {
	return RestoreDataFuncName
}

func validateAndGetOptArgs(args map[string]interface{}, tp param.TemplateParams) (string, string, string, map[string]string, string, string, bool, crv1alpha1.JSONMap, error) {
	var restorePath, encryptionKey, pod, tag, id string
	var vols map[string]string
	var podOverride crv1alpha1.JSONMap
	var err error
	var insecureTLS bool

	if err = OptArg(args, RestoreDataRestorePathArg, &restorePath, "/"); err != nil {
		return restorePath, encryptionKey, pod, vols, tag, id, insecureTLS, podOverride, err
	}
	if err = OptArg(args, RestoreDataEncryptionKeyArg, &encryptionKey, restic.GeneratePassword()); err != nil {
		return restorePath, encryptionKey, pod, vols, tag, id, insecureTLS, podOverride, err
	}
	if err = OptArg(args, RestoreDataPodArg, &pod, ""); err != nil {
		return restorePath, encryptionKey, pod, vols, tag, id, insecureTLS, podOverride, err
	}
	if err = OptArg(args, RestoreDataVolsArg, &vols, nil); err != nil {
		return restorePath, encryptionKey, pod, vols, tag, id, insecureTLS, podOverride, err
	}
	if (pod != "") == (len(vols) > 0) {
		return restorePath, encryptionKey, pod, vols, tag, id, insecureTLS, podOverride,
			errors.Errorf("Require one argument: %s or %s", RestoreDataPodArg, RestoreDataVolsArg)
	}
	if err = OptArg(args, RestoreDataBackupTagArg, &tag, nil); err != nil {
		return restorePath, encryptionKey, pod, vols, tag, id, insecureTLS, podOverride, err
	}
	if err = OptArg(args, RestoreDataBackupIdentifierArg, &id, nil); err != nil {
		return restorePath, encryptionKey, pod, vols, tag, id, insecureTLS, podOverride, err
	}
	if err = OptArg(args, InsecureTLS, &insecureTLS, false); err != nil {
		return restorePath, encryptionKey, pod, vols, tag, id, insecureTLS, podOverride, err
	}
	if (tag != "") == (id != "") {
		return restorePath, encryptionKey, pod, vols, tag, id, insecureTLS, podOverride,
			errors.Errorf("Require one argument: %s or %s", RestoreDataBackupTagArg, RestoreDataBackupIdentifierArg)
	}
	podOverride, err = GetPodSpecOverride(tp, args, RestoreDataPodOverrideArg)
	if err != nil {
		return restorePath, encryptionKey, pod, vols, tag, id, insecureTLS, podOverride, err
	}

	return restorePath, encryptionKey, pod, vols, tag, id, insecureTLS, podOverride, err
}

func restoreData(
	ctx context.Context,
	cli kubernetes.Interface,
	tp param.TemplateParams,
	namespace,
	encryptionKey,
	backupArtifactPrefix,
	restorePath,
	backupTag,
	backupID,
	jobPrefix,
	image string,
	insecureTLS bool,
	vols map[string]string,
	podOverride crv1alpha1.JSONMap,
	annotations,
	labels map[string]string,
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
		Namespace:    namespace,
		GenerateName: jobPrefix,
		Image:        image,
		Command:      []string{"sh", "-c", "tail -f /dev/null"},
		Volumes:      validatedVols,
		PodOverride:  podOverride,
		Annotations:  annotations,
		Labels:       labels,
	}

	// Apply the registered ephemeral pod changes.
	ephemeral.PodOptions.Apply(options)

	pr := kube.NewPodRunner(cli, options)
	podFunc := restoreDataPodFunc(tp, encryptionKey, backupArtifactPrefix, restorePath, backupTag, backupID, insecureTLS)
	return pr.Run(ctx, podFunc)
}

func restoreDataPodFunc(
	tp param.TemplateParams,
	encryptionKey,
	backupArtifactPrefix,
	restorePath,
	backupTag,
	backupID string,
	insecureTLS bool,
) func(ctx context.Context, pc kube.PodController) (map[string]interface{}, error) {
	return func(ctx context.Context, pc kube.PodController) (map[string]interface{}, error) {
		pod := pc.Pod()

		// Wait for pod to reach running state
		if err := pc.WaitForPodReady(ctx); err != nil {
			return nil, errors.Wrapf(err, "Failed while waiting for Pod %s to be ready", pod.Name)
		}

		remover, err := MaybeWriteProfileCredentials(ctx, pc, tp.Profile)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to write credentials to Pod %s", pc.PodName())
		}

		// Parent context could already be dead, so removing file within new context
		defer remover.Remove(context.Background()) //nolint:errcheck

		var cmd []string
		// Generate restore command based on the identifier passed
		if backupTag != "" {
			cmd, err = restic.RestoreCommandByTag(tp.Profile, backupArtifactPrefix, backupTag, restorePath, encryptionKey, insecureTLS)
		} else if backupID != "" {
			cmd, err = restic.RestoreCommandByID(tp.Profile, backupArtifactPrefix, backupID, restorePath, encryptionKey, insecureTLS)
		}
		if err != nil {
			return nil, err
		}

		ex, err := pc.GetCommandExecutor()
		if err != nil {
			return nil, err
		}
		var stdout, stderr bytes.Buffer
		err = ex.Exec(ctx, cmd, nil, &stdout, &stderr)
		format.LogWithCtx(ctx, pod.Name, pod.Spec.Containers[0].Name, stdout.String())
		format.LogWithCtx(ctx, pod.Name, pod.Spec.Containers[0].Name, stderr.String())
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to restore backup")
		}
		out, err := parseLogAndCreateOutput(stdout.String())
		return out, errors.Wrap(err, "Failed to parse phase output")
	}
}

func (r *restoreDataFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	// Set progress percent
	r.progressPercent = progress.StartedPercent
	defer func() { r.progressPercent = progress.CompletedPercent }()

	var namespace, image, backupArtifactPrefix, backupTag, backupID string
	var podOverride crv1alpha1.JSONMap
	var err error
	var bpAnnotations, bpLabels map[string]string
	if err = Arg(args, RestoreDataNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, RestoreDataImageArg, &image); err != nil {
		return nil, err
	}
	if err = Arg(args, RestoreDataBackupArtifactPrefixArg, &backupArtifactPrefix); err != nil {
		return nil, err
	}
	if err = OptArg(args, PodAnnotationsArg, &bpAnnotations, nil); err != nil {
		return nil, err
	}
	if err = OptArg(args, PodLabelsArg, &bpLabels, nil); err != nil {
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

	// Validate and get optional arguments
	restorePath, encryptionKey, pod, vols, backupTag, backupID, insecureTLS, podOverride, err := validateAndGetOptArgs(args, tp)
	if err != nil {
		return nil, err
	}
	if podOverride == nil {
		podOverride = tp.PodOverride
	}

	// Check if PodOverride specs are passed through actionset
	// If yes, override podOverride specs
	if tp.PodOverride != nil {
		podOverride, err = kube.CreateAndMergeJSONPatch(podOverride, tp.PodOverride)
		if err != nil {
			return nil, err
		}
	}

	if err = ValidateProfile(tp.Profile); err != nil {
		return nil, err
	}

	backupArtifactPrefix = ResolveArtifactPrefix(backupArtifactPrefix, tp.Profile)

	if len(vols) == 0 {
		// Fetch Volumes
		vols, err = FetchPodVolumes(pod, tp)
		if err != nil {
			return nil, err
		}
	}
	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	return restoreData(
		ctx,
		cli,
		tp,
		namespace,
		encryptionKey,
		backupArtifactPrefix,
		restorePath,
		backupTag,
		backupID,
		restoreDataJobPrefix,
		image,
		insecureTLS,
		vols,
		podOverride,
		annotations,
		labels,
	)
}

func (*restoreDataFunc) RequiredArgs() []string {
	return []string{
		RestoreDataNamespaceArg,
		RestoreDataImageArg,
		RestoreDataBackupArtifactPrefixArg,
	}
}

func (*restoreDataFunc) Arguments() []string {
	return []string{
		RestoreDataNamespaceArg,
		RestoreDataImageArg,
		RestoreDataBackupArtifactPrefixArg,
		RestoreDataRestorePathArg,
		RestoreDataEncryptionKeyArg,
		RestoreDataPodArg,
		RestoreDataVolsArg,
		RestoreDataBackupTagArg,
		RestoreDataBackupIdentifierArg,
		RestoreDataPodOverrideArg,
		InsecureTLS,
		PodAnnotationsArg,
		PodLabelsArg,
	}
}

func (r *restoreDataFunc) Validate(args map[string]any) error {
	if err := ValidatePodLabelsAndAnnotations(r.Name(), args); err != nil {
		return err
	}

	if err := utils.CheckSupportedArgs(r.Arguments(), args); err != nil {
		return err
	}

	return utils.CheckRequiredArgs(r.RequiredArgs(), args)
}

func (d *restoreDataFunc) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    d.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}
