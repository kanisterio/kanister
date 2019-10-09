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
	sp "k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/restic"
)

const (
	// DescribeBackupsNamespaceArg provides the namespace
	DescribeBackupsNamespaceArg = "namespace"
	// DescribeBackupsArtifactPrefixArg provides the path to restore backed up data
	DescribeBackupsArtifactPrefixArg = "backupArtifactPrefix"
	// DescribeBackupsEncryptionKeyArg provides the encryption key to be used for deletes
	DescribeBackupsEncryptionKeyArg = "encryptionKey"
	// DescribeBackupsPodOverrideArg contains pod specs to override default pod specs
	DescribeBackupsPodOverrideArg    = "podOverride"
	DescribeBackupsJobPrefix         = "get-backups-info-"
	DescribeBackupsFileCount         = "fileCount"
	DescribeBackupsSize              = "size"
	DescribeBackupsSnapshotIDs       = "snapshotIDs"
	DescribeBackupsPasswordIncorrect = "passwordIncorrect"
	DescribeBackupsRepoUnavailable   = "repoUnavailable"
	RawDataStatsMode                 = "raw-data"
)

func init() {
	kanister.Register(&DescribeBackupsFunc{})
}

var _ kanister.Func = (*DescribeBackupsFunc)(nil)

type DescribeBackupsFunc struct{}

func (*DescribeBackupsFunc) Name() string {
	return "DescribeBackups"
}

func describeBackups(ctx context.Context, cli kubernetes.Interface, tp param.TemplateParams, namespace, encryptionKey, targetPaths, jobPrefix string, podOverride sp.JSONMap) (map[string]interface{}, error) {
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
		pw, err := getPodWriter(cli, ctx, pod.Namespace, pod.Name, pod.Spec.Containers[0].Name, tp.Profile)
		if err != nil {
			return nil, err
		}
		defer cleanUpCredsFile(ctx, pw, pod.Namespace, pod.Name, pod.Spec.Containers[0].Name)
		snapshotIDs, err := restic.GetSnapshotIDs(tp.Profile, cli, targetPath, encryptionKey, namespace, pod.Name, pod.Spec.Containers[0].Name)
		if err != nil {
			if strings.Contains(err.Error(), restic.PasswordIncorrect) {
				return map[string]interface{}{
						DescribeBackupsSnapshotIDs:       nil,
						DescribeBackupsFileCount:         nil,
						DescribeBackupsSize:              nil,
						DescribeBackupsPasswordIncorrect: "true",
						DescribeBackupsRepoUnavailable:   nil,
					},
					nil
			}
			if strings.Contains(err.Error(), restic.RepoDoesNotExist) {
				return map[string]interface{}{
						DescribeBackupsSnapshotIDs:       nil,
						DescribeBackupsFileCount:         nil,
						DescribeBackupsSize:              nil,
						DescribeBackupsPasswordIncorrect: nil,
						DescribeBackupsRepoUnavailable:   "true",
					},
					nil
			}
			return nil, err
		}
		cmd, err := restic.StatsCommandByID(tp.Profile, targetPath, "" /* get all snapshot stats */, RawDataStatsMode, encryptionKey)
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
				DescribeBackupsSnapshotIDs:       snapshotIDs,
				DescribeBackupsFileCount:         fc,
				DescribeBackupsSize:              size,
				DescribeBackupsPasswordIncorrect: "false",
				DescribeBackupsRepoUnavailable:   "false",
			},
			nil
	}
}

func (*DescribeBackupsFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var namespace, getDescribeBackupsArtifactPrefix, encryptionKey string
	var err error
	if err = Arg(args, DescribeBackupsNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, DescribeBackupsArtifactPrefixArg, &getDescribeBackupsArtifactPrefix); err != nil {
		return nil, err
	}
	if err = OptArg(args, DescribeBackupsEncryptionKeyArg, &encryptionKey, restic.GeneratePassword()); err != nil {
		return nil, err
	}
	podOverride, err := GetPodSpecOverride(tp, args, DescribeBackupsPodOverrideArg)
	if err != nil {
		return nil, err
	}

	// Validate profile
	if err = validateProfile(tp.Profile); err != nil {
		return nil, err
	}
	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	return describeBackups(ctx, cli, tp, namespace, encryptionKey, getDescribeBackupsArtifactPrefix, DescribeBackupsJobPrefix, podOverride)
}

func (*DescribeBackupsFunc) RequiredArgs() []string {
	return []string{DescribeBackupsNamespaceArg, DescribeBackupsArtifactPrefixArg}
}
