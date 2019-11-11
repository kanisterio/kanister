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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/restic"
)

const (
	kanisterToolsImage = "kanisterio/kanister-tools:0.21.0"
	// CopyVolumeDataFuncName gives the function name
	CopyVolumeDataFuncName                     = "CopyVolumeData"
	copyVolumeDataMountPoint                   = "/mnt/vol_data/%s"
	copyVolumeDataJobPrefix                    = "copy-vol-data-"
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
	kanister.Register(&copyVolumeDataFunc{})
}

var _ kanister.Func = (*copyVolumeDataFunc)(nil)

type copyVolumeDataFunc struct{}

func (*copyVolumeDataFunc) Name() string {
	return CopyVolumeDataFuncName
}

func copyVolumeData(ctx context.Context, cli kubernetes.Interface, tp param.TemplateParams, namespace, pvc, targetPath, encryptionKey string, podOverride map[string]interface{}) (map[string]interface{}, error) {
	// Validate PVC exists
	if _, err := cli.CoreV1().PersistentVolumeClaims(namespace).Get(pvc, metav1.GetOptions{}); err != nil {
		return nil, errors.Wrapf(err, "Failed to retrieve PVC. Namespace %s, Name %s", namespace, pvc)
	}
	// Create a pod with PVCs attached
	mountPoint := fmt.Sprintf(copyVolumeDataMountPoint, pvc)
	options := &kube.PodOptions{
		Namespace:    namespace,
		GenerateName: copyVolumeDataJobPrefix,
		Image:        kanisterToolsImage,
		Command:      []string{"sh", "-c", "tail -f /dev/null"},
		Volumes:      map[string]string{pvc: mountPoint},
		PodOverride:  podOverride,
	}
	pr := kube.NewPodRunner(cli, options)
	podFunc := copyVolumeDataPodFunc(cli, tp, namespace, mountPoint, targetPath, encryptionKey)
	return pr.Run(ctx, podFunc)
}

func copyVolumeDataPodFunc(cli kubernetes.Interface, tp param.TemplateParams, namespace, mountPoint, targetPath, encryptionKey string) func(ctx context.Context, pod *v1.Pod) (map[string]interface{}, error) {
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
		stdout, stderr, err := kube.Exec(cli, namespace, pod.Name, pod.Spec.Containers[0].Name, cmd, nil)
		format.Log(pod.Name, pod.Spec.Containers[0].Name, stdout)
		format.Log(pod.Name, pod.Spec.Containers[0].Name, stderr)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to create and upload backup")
		}
		// Get the snapshot ID from log
		backupID := restic.SnapshotIDFromBackupLog(stdout)
		if backupID == "" {
			return nil, errors.New("Failed to parse the backup ID from logs")
		}
		fileCount, backupSize, phySize := restic.SnapshotStatsFromBackupLog(stdout)
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

	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	return copyVolumeData(ctx, cli, tp, namespace, vol, targetPath, encryptionKey, podOverride)
}

func (*copyVolumeDataFunc) RequiredArgs() []string {
	return []string{CopyVolumeDataNamespaceArg, CopyVolumeDataVolumeArg, CopyVolumeDataArtifactPrefixArg}
}
