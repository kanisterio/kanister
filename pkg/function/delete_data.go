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
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/ephemeral"
	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/restic"
	"github.com/kanisterio/kanister/pkg/utils"
)

const (
	// DeleteDataFuncName gives the function name
	DeleteDataFuncName = "DeleteData"
	// DeleteDataNamespaceArg provides the namespace
	DeleteDataNamespaceArg = "namespace"
	// DeleteDataBackupArtifactPrefixArg provides the path to restore backed up data
	DeleteDataBackupArtifactPrefixArg = "backupArtifactPrefix"
	// DeleteDataBackupIdentifierArg provides a unique ID added to the backed up artifacts
	DeleteDataBackupIdentifierArg = "backupID"
	// DeleteDataBackupTagArg provides a unique tag added to the backed up artifacts
	DeleteDataBackupTagArg = "backupTag"
	// DeleteDataEncryptionKeyArg provides the encryption key to be used for deletes
	DeleteDataEncryptionKeyArg = "encryptionKey"
	// DeleteDataReclaimSpace provides a way to specify if space should be reclaimed
	DeleteDataReclaimSpace = "reclaimSpace"
	// DeleteDataPodOverrideArg contains pod specs to override default pod specs
	DeleteDataPodOverrideArg = "podOverride"
	deleteDataJobPrefix      = "delete-data-"
	// DeleteDataOutputSpaceFreed is the key for the output reporting the space freed
	DeleteDataOutputSpaceFreed = "spaceFreed"
)

func init() {
	_ = kanister.Register(&deleteDataFunc{})
}

var _ kanister.Func = (*deleteDataFunc)(nil)

type deleteDataFunc struct {
	progressPercent string
}

func (*deleteDataFunc) Name() string {
	return DeleteDataFuncName
}

func deleteData(
	ctx context.Context,
	cli kubernetes.Interface,
	tp param.TemplateParams,
	reclaimSpace bool,
	namespace,
	encryptionKey string,
	insecureTLS bool,
	targetPaths,
	deleteTags,
	deleteIdentifiers []string,
	jobPrefix string,
	podOverride crv1alpha1.JSONMap,
	annotations,
	labels map[string]string,
) (map[string]interface{}, error) {
	if (len(deleteIdentifiers) == 0) == (len(deleteTags) == 0) {
		return nil, errors.Errorf("Require one argument: %s or %s", DeleteDataBackupIdentifierArg, DeleteDataBackupTagArg)
	}

	options := &kube.PodOptions{
		Namespace:    namespace,
		GenerateName: jobPrefix,
		Image:        consts.GetKanisterToolsImage(),
		Command:      []string{"sh", "-c", "tail -f /dev/null"},
		PodOverride:  podOverride,
		Annotations:  annotations,
		Labels:       labels,
	}

	// Apply the registered ephemeral pod changes.
	ephemeral.PodOptions.Apply(options)

	pr := kube.NewPodRunner(cli, options)
	podFunc := deleteDataPodFunc(tp, reclaimSpace, encryptionKey, insecureTLS, targetPaths, deleteTags, deleteIdentifiers)
	return pr.Run(ctx, podFunc)
}

//nolint:gocognit
func deleteDataPodFunc(
	tp param.TemplateParams,
	reclaimSpace bool,
	encryptionKey string,
	insecureTLS bool,
	targetPaths,
	deleteTags,
	deleteIdentifiers []string,
) func(ctx context.Context, pc kube.PodController) (map[string]interface{}, error) {
	return func(ctx context.Context, pc kube.PodController) (map[string]interface{}, error) {
		pod := pc.Pod()

		// Wait for pod to reach running state
		if err := pc.WaitForPodReady(ctx); err != nil {
			return nil, errors.Wrapf(err, "Failed while waiting for Pod %s to be ready", pod.Name)
		}

		remover, err := MaybeWriteProfileCredentials(ctx, pc, tp.Profile)
		if err != nil {
			return nil, err
		}

		// Parent context could already be dead, so removing file within new context
		defer remover.Remove(context.Background()) //nolint:errcheck

		// Get command executor
		podCommandExecutor, err := pc.GetCommandExecutor()
		if err != nil {
			return nil, err
		}

		for i, deleteTag := range deleteTags {
			cmd, err := restic.SnapshotsCommandByTag(tp.Profile, targetPaths[i], deleteTag, encryptionKey, insecureTLS)
			if err != nil {
				return nil, err
			}

			var stdout, stderr bytes.Buffer
			err = podCommandExecutor.Exec(ctx, cmd, nil, &stdout, &stderr)
			format.LogWithCtx(ctx, pod.Name, pod.Spec.Containers[0].Name, stdout.String())
			format.LogWithCtx(ctx, pod.Name, pod.Spec.Containers[0].Name, stderr.String())
			if err != nil {
				return nil, errors.Wrapf(err, "Failed to forget data, could not get snapshotID from tag, Tag: %s", deleteTag)
			}
			deleteIdentifier, err := restic.SnapshotIDFromSnapshotLog(stdout.String())
			if err != nil {
				return nil, errors.Wrapf(err, "Failed to forget data, could not get snapshotID from tag, Tag: %s", deleteTag)
			}
			deleteIdentifiers = append(deleteIdentifiers, deleteIdentifier)
		}
		var spaceFreedTotal int64
		for i, deleteIdentifier := range deleteIdentifiers {
			cmd, err := restic.ForgetCommandByID(tp.Profile, targetPaths[i], deleteIdentifier, encryptionKey, insecureTLS)
			if err != nil {
				return nil, err
			}

			var stdout, stderr bytes.Buffer
			err = podCommandExecutor.Exec(ctx, cmd, nil, &stdout, &stderr)
			format.LogWithCtx(ctx, pod.Name, pod.Spec.Containers[0].Name, stdout.String())
			format.LogWithCtx(ctx, pod.Name, pod.Spec.Containers[0].Name, stderr.String())
			if err != nil {
				return nil, errors.Wrapf(err, "Failed to forget data")
			}
			if reclaimSpace {
				spaceFreedStr, err := pruneData(tp, pod, podCommandExecutor, encryptionKey, targetPaths[i], insecureTLS)
				if err != nil {
					return nil, errors.Wrapf(err, "Error executing prune command")
				}
				spaceFreedTotal += restic.ParseResticSizeStringBytes(spaceFreedStr)
			}
		}

		return map[string]interface{}{
			DeleteDataOutputSpaceFreed: fmt.Sprintf("%d B", spaceFreedTotal),
		}, nil
	}
}

func pruneData(
	tp param.TemplateParams,
	pod *corev1.Pod,
	podCommandExecutor kube.PodCommandExecutor,
	encryptionKey,
	targetPath string,
	insecureTLS bool,
) (string, error) {
	cmd, err := restic.PruneCommand(tp.Profile, targetPath, encryptionKey, insecureTLS)
	if err != nil {
		return "", err
	}

	var stdout, stderr bytes.Buffer
	err = podCommandExecutor.Exec(context.Background(), cmd, nil, &stdout, &stderr)
	format.Log(pod.Name, pod.Spec.Containers[0].Name, stdout.String())
	format.Log(pod.Name, pod.Spec.Containers[0].Name, stderr.String())

	spaceFreed := restic.SpaceFreedFromPruneLog(stdout.String())
	return spaceFreed, errors.Wrapf(err, "Failed to prune data after forget")
}

func (d *deleteDataFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	// Set progress percent
	d.progressPercent = progress.StartedPercent
	defer func() { d.progressPercent = progress.CompletedPercent }()

	var namespace, deleteArtifactPrefix, deleteIdentifier, deleteTag, encryptionKey string
	var reclaimSpace bool
	var err error
	var insecureTLS bool
	var bpAnnotations, bpLabels map[string]string
	if err = Arg(args, DeleteDataNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, DeleteDataBackupArtifactPrefixArg, &deleteArtifactPrefix); err != nil {
		return nil, err
	}
	if err = OptArg(args, DeleteDataBackupIdentifierArg, &deleteIdentifier, ""); err != nil {
		return nil, err
	}
	if err = OptArg(args, DeleteDataBackupTagArg, &deleteTag, ""); err != nil {
		return nil, err
	}
	if err = OptArg(args, DeleteDataEncryptionKeyArg, &encryptionKey, restic.GeneratePassword()); err != nil {
		return nil, err
	}
	if err = OptArg(args, DeleteDataReclaimSpace, &reclaimSpace, false); err != nil {
		return nil, err
	}
	if err = OptArg(args, InsecureTLS, &insecureTLS, false); err != nil {
		return nil, err
	}
	if err = OptArg(args, PodAnnotationsArg, &bpAnnotations, nil); err != nil {
		return nil, err
	}
	if err = OptArg(args, PodLabelsArg, &bpLabels, nil); err != nil {
		return nil, err
	}

	podOverride, err := GetPodSpecOverride(tp, args, DeleteDataPodOverrideArg)
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

	if err = ValidateProfile(tp.Profile); err != nil {
		return nil, err
	}

	deleteArtifactPrefix = ResolveArtifactPrefix(deleteArtifactPrefix, tp.Profile)

	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	return deleteData(
		ctx,
		cli,
		tp,
		reclaimSpace,
		namespace,
		encryptionKey,
		insecureTLS,
		strings.Fields(deleteArtifactPrefix),
		strings.Fields(deleteTag),
		strings.Fields(deleteIdentifier),
		deleteDataJobPrefix,
		podOverride,
		annotations,
		labels,
	)
}

func (*deleteDataFunc) RequiredArgs() []string {
	return []string{
		DeleteDataNamespaceArg,
		DeleteDataBackupArtifactPrefixArg,
	}
}

func (*deleteDataFunc) Arguments() []string {
	return []string{
		DeleteDataNamespaceArg,
		DeleteDataBackupArtifactPrefixArg,
		DeleteDataBackupIdentifierArg,
		DeleteDataBackupTagArg,
		DeleteDataEncryptionKeyArg,
		DeleteDataReclaimSpace,
		InsecureTLS,
		PodAnnotationsArg,
		PodLabelsArg,
	}
}

func (d *deleteDataFunc) Validate(args map[string]any) error {
	if err := ValidatePodLabelsAndAnnotations(d.Name(), args); err != nil {
		return err
	}

	if err := utils.CheckSupportedArgs(d.Arguments(), args); err != nil {
		return err
	}

	return utils.CheckRequiredArgs(d.RequiredArgs(), args)
}

func (d *deleteDataFunc) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    d.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}
