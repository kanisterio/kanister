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

	"github.com/pkg/errors"
	"go.uber.org/zap/buffer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/restic"
)

const (
	// CopyVolumeDataFuncName gives the function name
	CopyVolumeDataFuncName                     = "CopyVolumeData"
	CopyVolumeDataMountPoint                   = "/mnt/vol_data/%s"
	CopyVolumeDataJobPrefix                    = "copy-vol-data-"
	CopyVolumeDataNamespaceArg                 = "namespace"
	CopyVolumeDataVolumeArg                    = "volume"
	CopyVolumeDataArtifactPrefixArg            = "dataArtifactPrefix"
	CopyVolumeDataOutputBackupID               = "backupID"
	CopyVolumeDataOutputBackupRoot             = "backupRoot"
	CopyVolumeDataOutputBackupArtifactLocation = "backupArtifactLocation"
	CopyVolumeDataEncryptionKeyArg             = "encryptionKey"
	CopyVolumeDataOutputBackupTag              = "backupTag"
	CopyVolumeDataPodOverrideArg               = "podOverride"
	CopyVolumeDataOutputBackupFileCount        = "fileCount"
	CopyVolumeDataOutputBackupSize             = "size"
	CopyVolumeDataOutputPhysicalSize           = "phySize"
)

func init() {
	_ = kanister.Register(&copyVolumeDataFunc{})
}

var _ kanister.Func = (*copyVolumeDataFunc)(nil)

type copyVolumeDataFunc struct{}

func (*copyVolumeDataFunc) Name() string {
	return CopyVolumeDataFuncName
}

func copyVolumeData(ctx context.Context, cli kubernetes.Interface, tp param.TemplateParams, namespace, pvc, targetPath, encryptionKey string, podOverride map[string]interface{}) (map[string]interface{}, error) {
	// Validate PVC exists
	if _, err := cli.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, pvc, metav1.GetOptions{}); err != nil {
		return nil, errors.Wrapf(err, "Failed to retrieve PVC. Namespace %s, Name %s", namespace, pvc)
	}
	// Create a pod with PVCs attached
	mountPoint := fmt.Sprintf(CopyVolumeDataMountPoint, pvc)
	options := &kube.PodOptions{
		Namespace:    namespace,
		GenerateName: CopyVolumeDataJobPrefix,
		Image:        consts.GetKanisterToolsImage(),
		Command:      []string{"sh", "-c", "tail -f /dev/null"},
		Volumes:      map[string]string{pvc: mountPoint},
		PodOverride:  podOverride,
	}
	pr := kube.NewPodRunner(cli, options)
	podFunc := copyVolumeDataPodFunc(cli, tp, namespace, mountPoint, targetPath, encryptionKey)
	return pr.RunEx(ctx, podFunc)
}

func copyVolumeDataPodFunc(cli kubernetes.Interface, tp param.TemplateParams, namespace, mountPoint, targetPath, encryptionKey string) func(ctx context.Context, pc kube.PodController) (map[string]interface{}, error) {
	return func(ctx context.Context, pc kube.PodController) (map[string]interface{}, error) {
		// Wait for pod to reach running state
		if err := pc.WaitForPodReady(ctx); err != nil {
			return nil, errors.Wrapf(err, "Failed while waiting for Pod %s to be ready", pc.PodName())
		}
		pw1, err := pc.GetFileWriter()
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to write credentials to Pod %s", pc.PodName())
		}

		remover, err := WriteCredsToPod(ctx, pw1, tp.Profile)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to write credentials to Pod %s", pc.PodName())
		}

		// Parent context could already be dead, so removing file within new context
		defer remover.Remove(context.Background()) //nolint:errcheck

		pod := pc.Pod()
		// Get restic repository
		if err := restic.GetOrCreateRepository(cli, namespace, pod.Name, pod.Spec.Containers[0].Name, targetPath, encryptionKey, tp.Profile); err != nil {
			return nil, err
		}
		// Copy data to object store
		backupTag := rand.String(10)
		cmd, err := restic.BackupCommandByTag(tp.Profile, targetPath, backupTag, mountPoint, encryptionKey)
		if err != nil {
			return nil, err
		}
		ex, err := pc.GetCommandExecutor()
		if err != nil {
			return nil, err
		}
		var stdout buffer.Buffer
		var stderr buffer.Buffer
		err = ex.Exec(ctx, cmd, nil, &stdout, &stderr)
		format.LogWithCtx(ctx, pod.Name, pod.Spec.Containers[0].Name, stdout.String())
		format.LogWithCtx(ctx, pod.Name, pod.Spec.Containers[0].Name, stderr.String())
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to create and upload backup")
		}
		// Get the snapshot ID from log
		backupID := restic.SnapshotIDFromBackupLog(stdout.String())
		if backupID == "" {
			return nil, errors.Errorf("Failed to parse the backup ID from logs, backup logs %s", stdout.String())
		}
		fileCount, backupSize, phySize := restic.SnapshotStatsFromBackupLog(stdout.String())
		if backupSize == "" {
			log.Debug().Print("Could not parse backup stats from backup log")
		}
		return map[string]interface{}{
				CopyVolumeDataOutputBackupID:               backupID,
				CopyVolumeDataOutputBackupRoot:             mountPoint,
				CopyVolumeDataOutputBackupArtifactLocation: targetPath,
				CopyVolumeDataOutputBackupTag:              backupTag,
				CopyVolumeDataOutputBackupFileCount:        fileCount,
				CopyVolumeDataOutputBackupSize:             backupSize,
				CopyVolumeDataOutputPhysicalSize:           phySize,
				FunctionOutputVersion:                      kanister.DefaultVersion,
			},
			nil
	}
}

func (*copyVolumeDataFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var namespace, vol, targetPath, encryptionKey string
	var err error
	if err = Arg(args, CopyVolumeDataNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, CopyVolumeDataVolumeArg, &vol); err != nil {
		return nil, err
	}
	if err = Arg(args, CopyVolumeDataArtifactPrefixArg, &targetPath); err != nil {
		return nil, err
	}
	if err = OptArg(args, CopyVolumeDataEncryptionKeyArg, &encryptionKey, restic.GeneratePassword()); err != nil {
		return nil, err
	}
	podOverride, err := GetPodSpecOverride(tp, args, CopyVolumeDataPodOverrideArg)
	if err != nil {
		return nil, err
	}

	if err = ValidateProfile(tp.Profile); err != nil {
		return nil, errors.Wrapf(err, "Failed to validate Profile")
	}

	targetPath = ResolveArtifactPrefix(targetPath, tp.Profile)

	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	return copyVolumeData(ctx, cli, tp, namespace, vol, targetPath, encryptionKey, podOverride)
}

func (*copyVolumeDataFunc) RequiredArgs() []string {
	return []string{
		CopyVolumeDataNamespaceArg,
		CopyVolumeDataVolumeArg,
		CopyVolumeDataArtifactPrefixArg,
	}
}

func (*copyVolumeDataFunc) Arguments() []string {
	return []string{
		CopyVolumeDataNamespaceArg,
		CopyVolumeDataVolumeArg,
		CopyVolumeDataArtifactPrefixArg,
		CopyVolumeDataEncryptionKeyArg,
	}
}
