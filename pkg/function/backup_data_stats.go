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
	backupDataStatsJobPrefix = "backup-data-stats-"
	// BackupDataStatsFuncName gives the name of the stats function
	BackupDataStatsFuncName = "BackupDataStats"
	// BackupDataStatsNamespaceArg provides the namespace
	BackupDataStatsNamespaceArg = "namespace"
	// BackupDataStatsBackupArtifactPrefixArg provides the path to store artifacts on the object store
	BackupDataStatsBackupArtifactPrefixArg = "backupArtifactPrefix"
	// BackupDataStatsEncryptionKeyArg provides the encryption key to be used for backups
	BackupDataStatsEncryptionKeyArg = "encryptionKey"
	// BackupDataStatsBackupIdentifierArg provides a unique ID added to the backed up artifacts
	BackupDataStatsBackupIdentifierArg = "backupID"
	// BackupDataStatsMode provides a mode for stats
	BackupDataStatsMode            = "statsMode"
	BackupDataStatsOutputFileCount = "fileCount"
	BackupDataStatsOutputSize      = "size"
	BackupDataStatsOutputMode      = "mode"
	defaultStatsMode               = "restore-size"
)

func init() {
	_ = kanister.Register(&BackupDataStatsFunc{})
}

var _ kanister.Func = (*BackupDataStatsFunc)(nil)

type BackupDataStatsFunc struct {
	progressPercent string
}

func (*BackupDataStatsFunc) Name() string {
	return BackupDataStatsFuncName
}

func backupDataStats(
	ctx context.Context,
	cli kubernetes.Interface,
	tp param.TemplateParams,
	namespace,
	encryptionKey,
	backupArtifactPrefix,
	backupID,
	mode,
	jobPrefix string,
	podOverride crv1alpha1.JSONMap,
	annotations,
	labels map[string]string,
) (map[string]interface{}, error) {
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
	podFunc := backupDataStatsPodFunc(tp, encryptionKey, backupArtifactPrefix, backupID, mode)
	return pr.Run(ctx, podFunc)
}

func backupDataStatsPodFunc(
	tp param.TemplateParams,
	encryptionKey,
	backupArtifactPrefix,
	backupID,
	mode string,
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

		cmd, err := restic.StatsCommandByID(tp.Profile, backupArtifactPrefix, backupID, mode, encryptionKey)
		if err != nil {
			return nil, err
		}

		commandExecutor, err := pc.GetCommandExecutor()
		if err != nil {
			return nil, errors.Wrap(err, "Unable to get pod command executor")
		}

		var stdout, stderr bytes.Buffer
		err = commandExecutor.Exec(ctx, cmd, nil, &stdout, &stderr)
		format.LogWithCtx(ctx, pod.Name, pod.Spec.Containers[0].Name, stdout.String())
		format.LogWithCtx(ctx, pod.Name, pod.Spec.Containers[0].Name, stderr.String())
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to get backup stats")
		}
		// Get File Count and Size from Stats
		mode, fc, size := restic.SnapshotStatsFromStatsLog(stdout.String())
		if fc == "" || size == "" {
			return nil, errors.New("Failed to parse snapshot stats from logs")
		}
		return map[string]interface{}{
				BackupDataStatsOutputMode:      mode,
				BackupDataStatsOutputFileCount: fc,
				BackupDataStatsOutputSize:      size,
				FunctionOutputVersion:          kanister.DefaultVersion,
			},
			nil
	}
}

func (b *BackupDataStatsFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	// Set progress percent
	b.progressPercent = progress.StartedPercent
	defer func() { b.progressPercent = progress.CompletedPercent }()

	var namespace, backupArtifactPrefix, backupID, mode, encryptionKey string
	var err error
	var bpAnnotations, bpLabels map[string]string
	if err = Arg(args, BackupDataStatsNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, BackupDataStatsBackupArtifactPrefixArg, &backupArtifactPrefix); err != nil {
		return nil, err
	}
	if err = Arg(args, BackupDataStatsBackupIdentifierArg, &backupID); err != nil {
		return nil, err
	}
	if err = OptArg(args, BackupDataStatsMode, &mode, defaultStatsMode); err != nil {
		return nil, err
	}
	if err = OptArg(args, BackupDataStatsEncryptionKeyArg, &encryptionKey, restic.GeneratePassword()); err != nil {
		return nil, err
	}
	if err = OptArg(args, PodAnnotationsArg, &bpAnnotations, nil); err != nil {
		return nil, err
	}
	if err = OptArg(args, PodLabelsArg, &bpLabels, nil); err != nil {
		return nil, err
	}

	podOverride, err := GetPodSpecOverride(tp, args, CheckRepositoryPodOverrideArg)
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
		return nil, errors.Wrapf(err, "Failed to validate Profile")
	}

	backupArtifactPrefix = ResolveArtifactPrefix(backupArtifactPrefix, tp.Profile)

	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	return backupDataStats(
		ctx,
		cli,
		tp,
		namespace,
		encryptionKey,
		backupArtifactPrefix,
		backupID,
		mode,
		backupDataStatsJobPrefix,
		podOverride,
		annotations,
		labels,
	)
}

func (*BackupDataStatsFunc) RequiredArgs() []string {
	return []string{
		BackupDataStatsNamespaceArg,
		BackupDataStatsBackupArtifactPrefixArg,
		BackupDataStatsBackupIdentifierArg,
	}
}

func (*BackupDataStatsFunc) Arguments() []string {
	return []string{
		BackupDataStatsNamespaceArg,
		BackupDataStatsBackupArtifactPrefixArg,
		BackupDataStatsBackupIdentifierArg,
		BackupDataStatsMode,
		BackupDataStatsEncryptionKeyArg,
		PodAnnotationsArg,
		PodLabelsArg,
	}
}

func (b *BackupDataStatsFunc) Validate(args map[string]any) error {
	if err := ValidatePodLabelsAndAnnotations(b.Name(), args); err != nil {
		return err
	}

	if err := utils.CheckSupportedArgs(b.Arguments(), args); err != nil {
		return err
	}

	return utils.CheckRequiredArgs(b.RequiredArgs(), args)
}

func (b *BackupDataStatsFunc) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    b.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}
