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
	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/restic"
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
	kanister.Register(&BackupDataStatsFunc{})
}

var _ kanister.Func = (*BackupDataStatsFunc)(nil)

type BackupDataStatsFunc struct{}

func (*BackupDataStatsFunc) Name() string {
	return BackupDataStatsFuncName
}

func backupDataStats(ctx context.Context, cli kubernetes.Interface, tp param.TemplateParams, namespace, encryptionKey, backupArtifactPrefix, backupID, mode, jobPrefix string) (map[string]interface{}, error) {
	options := &kube.PodOptions{
		Namespace:    namespace,
		GenerateName: jobPrefix,
		Image:        kanisterToolsImage,
		Command:      []string{"sh", "-c", "tail -f /dev/null"},
	}
	pr := kube.NewPodRunner(cli, options)
	podFunc := backupDataStatsPodFunc(cli, tp, namespace, encryptionKey, backupArtifactPrefix, backupID, mode)
	return pr.Run(ctx, podFunc)
}

func backupDataStatsPodFunc(cli kubernetes.Interface, tp param.TemplateParams, namespace, encryptionKey, backupArtifactPrefix, backupID, mode string) func(ctx context.Context, pod *v1.Pod) (map[string]interface{}, error) {
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
		cmd, err := restic.StatsCommandByID(tp.Profile, backupArtifactPrefix, backupID, mode, encryptionKey)
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
		mode, fc, size := restic.SnapshotStatsFromStatsLog(stdout)
		if fc == "" || size == "" {
			return nil, errors.New("Failed to parse snapshot stats from logs")
		}
		return map[string]interface{}{
				BackupDataStatsOutputMode:      mode,
				BackupDataStatsOutputFileCount: fc,
				BackupDataStatsOutputSize:      size,
			},
			nil
	}
}

func (*BackupDataStatsFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var namespace, backupArtifactPrefix, backupID, mode, encryptionKey string
	var err error
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
	if err = ValidateProfile(tp.Profile); err != nil {
		return nil, errors.Wrapf(err, "Failed to validate Profile")
	}
	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	return backupDataStats(ctx, cli, tp, namespace, encryptionKey, backupArtifactPrefix, backupID, mode, backupDataStatsJobPrefix)
}

func (*BackupDataStatsFunc) RequiredArgs() []string {
	return []string{BackupDataStatsNamespaceArg, BackupDataStatsBackupArtifactPrefixArg}
}
