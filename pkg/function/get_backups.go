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
	sp "k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/restic"
)

const (
	// getBackupsNamespaceArg provides the namespace
	getBackupsNamespaceArg = "namespace"
	// getBackupsArtifactPrefixArg provides the path to restore backed up data
	getBackupsArtifactPrefixArg = "backupArtifactPrefix"
	// getBackupsEncryptionKeyArg provides the encryption key to be used for deletes
	getBackupsEncryptionKeyArg = "encryptionKey"
	// getBackupsPodOverrideArg contains pod specs to override default pod specs
	getBackupsPodOverrideArg = "podOverride"
	getBackupsJobPrefix      = "get-backups-data-"
	GetBackupsFileCount      = "filecount"
	GetBackupsSize           = "size"
	GetBackupsSnapshotIDs    = "snapshotIDs"
)

func init() {
	kanister.Register(&getBackupsFunc{})
}

var _ kanister.Func = (*getBackupsFunc)(nil)

type getBackupsFunc struct{}

func (*getBackupsFunc) Name() string {
	return "getBackups"
}

func getBackups(ctx context.Context, cli kubernetes.Interface, tp param.TemplateParams, namespace, encryptionKey, targetPaths, jobPrefix string, podOverride sp.JSONMap) (map[string]interface{}, error) {
	options := &kube.PodOptions{
		Namespace:    namespace,
		GenerateName: jobPrefix,
		Image:        kanisterToolsImage,
		Command:      []string{"sh", "-c", "tail -f /dev/null"},
		PodOverride:  podOverride,
	}
	pr := kube.NewPodRunner(cli, options)
	podFunc := getBackupsPodFunc(cli, tp, namespace, encryptionKey, targetPaths)
	return pr.Run(ctx, podFunc)
}

func getBackupsPodFunc(cli kubernetes.Interface, tp param.TemplateParams, namespace, encryptionKey, targetPath string) func(ctx context.Context, pod *v1.Pod) (map[string]interface{}, error) {
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
			return nil, err
		}
		cmd, err := restic.StatsCommandByID(tp.Profile, targetPath, "" /* get all snapshot stats */, DefaultStatsMode, encryptionKey)
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
				GetBackupsSnapshotIDs: snapshotIDs,
				GetBackupsFileCount:   fc,
				GetBackupsSize:        size,
			},
			nil
	}
}

func (*getBackupsFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var namespace, getBackupsArtifactPrefix, encryptionKey string
	var err error
	if err = Arg(args, getBackupsNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, getBackupsArtifactPrefixArg, &getBackupsArtifactPrefix); err != nil {
		return nil, err
	}
	if err = OptArg(args, getBackupsEncryptionKeyArg, &encryptionKey, restic.GeneratePassword()); err != nil {
		return nil, err
	}
	podOverride, err := GetPodSpecOverride(tp, args, getBackupsPodOverrideArg)
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
	return getBackups(ctx, cli, tp, namespace, encryptionKey, getBackupsArtifactPrefix, getBackupsJobPrefix, podOverride)
}

func (*getBackupsFunc) RequiredArgs() []string {
	return []string{getBackupsNamespaceArg, getBackupsArtifactPrefixArg}
}
