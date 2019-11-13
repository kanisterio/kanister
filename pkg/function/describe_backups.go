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

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
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
	rawDataStatsMode                 = "raw-data"
)

func init() {
	kanister.Register(&DescribeBackupsFunc{})
}

var _ kanister.Func = (*DescribeBackupsFunc)(nil)

type DescribeBackupsFunc struct{}

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
		Image:        kanisterToolsImage,
		Command:      []string{"sh", "-c", "tail -f /dev/null"},
		PodOverride:  podOverride,
	}
	pr := kube.NewPodRunner(cli, options)
	podFunc := describeBackupsPodFunc(cli, tp, namespace, encryptionKey, targetPaths)
	return pr.Run(ctx, podFunc)
}

func describeBackupsPodFunc(cli kubernetes.Interface, tp param.TemplateParams, namespace, encryptionKey, targetPath string) func(ctx context.Context, pod *v1.Pod) (map[string]interface{}, error) {
	return func(ctx context.Context, pod *v1.Pod) (map[string]interface{}, error) {
		// Wait for pod to reach running state
		if err := kube.WaitForPodReady(ctx, cli, pod.Namespace, pod.Name); err != nil {
			return nil, errors.Wrapf(err, "Failed while waiting for Pod %s to be ready", pod.Name)
		}
		pw, err := GetPodWriter(cli, ctx, pod.Namespace, pod.Name, pod.Spec.Containers[0].Name, tp.Profile)
		if err != nil {
			return nil, err
		}
		defer CleanUpCredsFile(ctx, pw, pod.Namespace, pod.Name, pod.Spec.Containers[0].Name)
		err = restic.CheckIfRepoIsReachable(tp.Profile, targetPath, encryptionKey, cli, namespace, pod.Name, pod.Spec.Containers[0].Name)
		switch {
		case err == nil:
			break
		case strings.Contains(err.Error(), restic.PasswordIncorrect):
			return map[string]interface{}{
					DescribeBackupsFileCount:         nil,
					DescribeBackupsSize:              nil,
					DescribeBackupsPasswordIncorrect: "true",
					DescribeBackupsRepoDoesNotExist:  "false",
				},
				nil

		case strings.Contains(err.Error(), restic.RepoDoesNotExist):
			return map[string]interface{}{
					DescribeBackupsFileCount:         nil,
					DescribeBackupsSize:              nil,
					DescribeBackupsPasswordIncorrect: "false",
					DescribeBackupsRepoDoesNotExist:  "true",
				},
				nil
		default:
			return nil, err

		}
		cmd, err := restic.StatsCommandByID(tp.Profile, targetPath, "" /* get all snapshot stats */, rawDataStatsMode, encryptionKey)
		if err != nil {
			return nil, err
		}
		stdout, stderr, err := kube.Exec(cli, namespace, pod.Name, pod.Spec.Containers[0].Name, cmd, nil)
		format.Log(pod.Name, pod.Spec.Containers[0].Name, stdout)
		format.Log(pod.Name, pod.Spec.Containers[0].Name, stderr)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to get backup stats")
		}
		// Get File Count and Size from Stats
		_, fc, size := restic.SnapshotStatsFromStatsLog(stdout)
		if fc == "" || size == "" {
			return nil, errors.New("Failed to parse snapshot stats from logs")
		}
		return map[string]interface{}{
				DescribeBackupsFileCount:         fc,
				DescribeBackupsSize:              size,
				DescribeBackupsPasswordIncorrect: "false",
				DescribeBackupsRepoDoesNotExist:  "false",
			},
			nil
	}
}

func (*DescribeBackupsFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
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
