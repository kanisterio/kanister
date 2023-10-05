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
	"strings"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/restic"
)

const (
	// DescribeBackupsFuncName gives the function name
	DescribeBackupsFuncName = "DescribeBackups"
	// DescribeBackupsArtifactPrefixArg provides the path to restore backed up data
	DescribeBackupsArtifactPrefixArg = "backupArtifactPrefix"
	// DescribeBackupsEncryptionKeyArg provides the encryption key to be used for deletes
	DescribeBackupsEncryptionKeyArg = "encryptionKey"
	// DescribeBackupsPodOverrideArg contains pod specs to override default pod specs
	DescribeBackupsPodOverrideArg    = "podOverride"
	DescribeBackupsJobPrefix         = "describe-backups-"
	DescribeBackupsFileCount         = "fileCount"
	DescribeBackupsSize              = "size"
	DescribeBackupsPasswordIncorrect = "passwordIncorrect"
	DescribeBackupsRepoDoesNotExist  = "repoUnavailable"
	RawDataStatsMode                 = "raw-data"
)

func init() {
	_ = kanister.Register(&DescribeBackupsFunc{})
}

var _ kanister.Func = (*DescribeBackupsFunc)(nil)

type DescribeBackupsFunc struct {
	progressPercent string
}

func (*DescribeBackupsFunc) Name() string {
	return DescribeBackupsFuncName
}

func describeBackups(ctx context.Context, cli kubernetes.Interface, tp param.TemplateParams, encryptionKey, targetPaths, jobPrefix string, podOverride crv1alpha1.JSONMap) (map[string]interface{}, error) {
	namespace, err := kube.GetControllerNamespace()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get controller namespace")
	}
	options := &kube.PodOptions{
		Namespace:    namespace,
		GenerateName: jobPrefix,
		Image:        consts.GetKanisterToolsImage(),
		Command:      []string{"sh", "-c", "tail -f /dev/null"},
		PodOverride:  podOverride,
	}
	pr := kube.NewPodRunner(cli, options)
	podFunc := describeBackupsPodFunc(cli, tp, encryptionKey, targetPaths)
	return pr.Run(ctx, podFunc)
}

func describeBackupsPodFunc(
	cli kubernetes.Interface,
	tp param.TemplateParams,
	encryptionKey,
	targetPath string,
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

		err = restic.CheckIfRepoIsReachable(
			tp.Profile,
			targetPath,
			encryptionKey,
			cli, pod.Namespace,
			pod.Name,
			pod.Spec.Containers[0].Name,
		)
		switch {
		case err == nil:
			break
		case strings.Contains(err.Error(), restic.PasswordIncorrect):
			return map[string]interface{}{
					DescribeBackupsFileCount:         nil,
					DescribeBackupsSize:              nil,
					DescribeBackupsPasswordIncorrect: "true",
					DescribeBackupsRepoDoesNotExist:  "false",
					FunctionOutputVersion:            kanister.DefaultVersion,
				},
				nil

		case strings.Contains(err.Error(), restic.RepoDoesNotExist):
			return map[string]interface{}{
					DescribeBackupsFileCount:         nil,
					DescribeBackupsSize:              nil,
					DescribeBackupsPasswordIncorrect: "false",
					DescribeBackupsRepoDoesNotExist:  "true",
					FunctionOutputVersion:            kanister.DefaultVersion,
				},
				nil
		default:
			return nil, err
		}

		cmd, err := restic.StatsCommandByID(tp.Profile, targetPath, "" /* get all snapshot stats */, RawDataStatsMode, encryptionKey)
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
			return nil, errors.Wrapf(err, "Failed to get backup stats")
		}

		// Get File Count and Size from Stats
		_, fc, size := restic.SnapshotStatsFromStatsLog(stdout.String())
		if fc == "" || size == "" {
			return nil, errors.New("Failed to parse snapshot stats from logs")
		}
		return map[string]interface{}{
				DescribeBackupsFileCount:         fc,
				DescribeBackupsSize:              size,
				DescribeBackupsPasswordIncorrect: "false",
				DescribeBackupsRepoDoesNotExist:  "false",
				FunctionOutputVersion:            kanister.DefaultVersion,
			},
			nil
	}
}

func (d *DescribeBackupsFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	// Set progress percent
	d.progressPercent = progress.StartedPercent
	defer func() { d.progressPercent = progress.CompletedPercent }()

	var describeBackupsArtifactPrefix, encryptionKey string
	var err error
	if err = Arg(args, DescribeBackupsArtifactPrefixArg, &describeBackupsArtifactPrefix); err != nil {
		return nil, err
	}
	if err = OptArg(args, DescribeBackupsEncryptionKeyArg, &encryptionKey, restic.GeneratePassword()); err != nil {
		return nil, err
	}
	podOverride, err := GetPodSpecOverride(tp, args, DescribeBackupsPodOverrideArg)
	if err != nil {
		return nil, err
	}

	if err = ValidateProfile(tp.Profile); err != nil {
		return nil, err
	}

	describeBackupsArtifactPrefix = ResolveArtifactPrefix(describeBackupsArtifactPrefix, tp.Profile)

	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	return describeBackups(ctx, cli, tp, encryptionKey, describeBackupsArtifactPrefix, DescribeBackupsJobPrefix, podOverride)
}

func (*DescribeBackupsFunc) RequiredArgs() []string {
	return []string{DescribeBackupsArtifactPrefixArg}
}

func (*DescribeBackupsFunc) Arguments() []string {
	return []string{
		DescribeBackupsArtifactPrefixArg,
		DescribeBackupsEncryptionKeyArg,
	}
}

func (d *DescribeBackupsFunc) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    d.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}
