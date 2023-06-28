// Copyright 2023 The Kanister Authors.
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
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/format"
	kankopia "github.com/kanisterio/kanister/pkg/kopia"
	kopiacmd "github.com/kanisterio/kanister/pkg/kopia/command"
	kerrors "github.com/kanisterio/kanister/pkg/kopia/errors"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/utils"
)

const (
	CopyVolumeDataUsingKopiaServerFuncName = "CopyVolumeDataUsingKopiaServer"
	// CopyVolumeDataUsingKopiaServerSnapshotTagsArg is the key used for returning snapshot tags
	CopyVolumeDataUsingKopiaServerSnapshotTagsArg = "snapshotTags"
)

type copyVolumeDataUsingKopiaServerFunc struct{}

func init() {
	err := kanister.Register(&copyVolumeDataUsingKopiaServerFunc{})
	if err != nil {
		return
	}
}

var _ kanister.Func = (*copyVolumeDataUsingKopiaServerFunc)(nil)

func (*copyVolumeDataUsingKopiaServerFunc) Name() string {
	return CopyVolumeDataUsingKopiaServerFuncName
}

func (f *copyVolumeDataUsingKopiaServerFunc) RequiredArgs() []string {
	return []string{
		CopyVolumeDataNamespaceArg,
		CopyVolumeDataVolumeArg}
}

func (f *copyVolumeDataUsingKopiaServerFunc) Arguments() []string {
	return []string{
		CopyVolumeDataNamespaceArg,
		CopyVolumeDataVolumeArg,
		CopyVolumeDataEncryptionKeyArg,
		CopyVolumeDataUsingKopiaServerSnapshotTagsArg,
	}
}

func (f *copyVolumeDataUsingKopiaServerFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]any) (map[string]any, error) {
	var namespace, vol, encryptionKey, tagsStr string
	var err error
	if err = Arg(args, CopyVolumeDataNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, CopyVolumeDataVolumeArg, &vol); err != nil {
		return nil, err
	}
	if err = Arg(args, CopyVolumeDataEncryptionKeyArg, &encryptionKey); err != nil {
		return nil, err
	}
	if err = OptArg(args, CopyVolumeDataUsingKopiaServerSnapshotTagsArg, &tagsStr, ""); err != nil {
		return nil, err
	}

	var tags []string = nil
	if tagsStr != "" {
		tags = strings.Split(tagsStr, ",")
	}

	podOverride, err := GetPodSpecOverride(tp, args, CopyVolumeDataPodOverrideArg)
	if err != nil {
		return nil, err
	}

	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	return copyVolumeDataUsingKopiaServer(ctx, cli, tp, namespace, vol, encryptionKey, podOverride, tags)
}

func copyVolumeDataUsingKopiaServer(ctx context.Context, cli kubernetes.Interface, tp param.TemplateParams, namespace, pvc, encryptionKey string, podOverride map[string]interface{}, tags []string) (map[string]interface{}, error) {
	// Validate PVC exists
	if _, err := cli.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, pvc, metav1.GetOptions{}); err != nil {
		return nil, errors.Wrapf(err, "Failed to retrieve PVC. Namespace %s, Name %s", namespace, pvc)
	}
	// Create a pod with PVCs attached
	mountPoint := fmt.Sprintf(CopyVolumeDataMountPoint, pvc)
	options := &kube.PodOptions{
		Namespace:    namespace,
		GenerateName: CopyVolumeDataJobPrefix,
		Image:        getKanisterToolsImage(),
		Command:      []string{"sh", "-c", "tail -f /dev/null"},
		Volumes:      map[string]string{pvc: mountPoint},
		PodOverride:  podOverride,
	}
	pr := kube.NewPodRunner(cli, options)
	podFunc := copyVolumeDataUsingKopiaServerPodFunc(cli, tp, namespace, mountPoint, encryptionKey, tags)
	return pr.Run(ctx, podFunc)
}

func copyVolumeDataUsingKopiaServerPodFunc(cli kubernetes.Interface, tp param.TemplateParams, namespace, mountPoint, encryptionKey string, tags []string) func(ctx context.Context, pod *corev1.Pod) (map[string]any, error) {
	return func(ctx context.Context, pod *corev1.Pod) (map[string]any, error) {
		if err := kube.WaitForPodReady(ctx, cli, pod.Namespace, pod.Name); err != nil {
			return nil, errors.Wrap(err, "Failed while waiting for Pod: "+pod.Name+" to be ready")
		}
		userPassphrase, cert, err := userCredentialsAndServerTLS(&tp)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to fetch User Credentials/Certificate Data from Template Params")
		}

		fingerprint, err := kankopia.ExtractFingerprintFromCertificateJSON(cert)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to fetch Kopia API Server Certificate Secret Data from Certificate")
		}

		hostname, userAccessPassphrase, err := hostNameAndUserPassPhraseFromRepoServer(userPassphrase)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to fetch Hostname/User Passphrase from Secret")
		}

		configFile, logDirectory := kankopia.CustomConfigFileAndLogDirectory(hostname)

		cmd := kopiacmd.RepositoryConnectServerCommand(kopiacmd.RepositoryServerCommandArgs{
			UserPassword:    userAccessPassphrase,
			ConfigFilePath:  configFile,
			LogDirectory:    logDirectory,
			CacheDirectory:  kopiacmd.DefaultCacheDirectory,
			Hostname:        hostname,
			ServerURL:       tp.RepositoryServer.Address,
			Fingerprint:     fingerprint,
			Username:        tp.RepositoryServer.Username,
			ContentCacheMB:  tp.RepositoryServer.ContentCacheMB,
			MetadataCacheMB: tp.RepositoryServer.MetadataCacheMB,
		})
		stdout, stderr, err := kube.Exec(cli, namespace, pod.Name, pod.Spec.Containers[0].Name, cmd, nil)
		format.Log(pod.Name, pod.Spec.Containers[0].Name, stdout)
		format.Log(pod.Name, pod.Spec.Containers[0].Name, stderr)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to connect to Kopia API server")
		}

		cmd = kopiacmd.SnapshotCreate(
			kopiacmd.SnapshotCreateCommandArgs{
				PathToBackup: mountPoint,
				CommandArgs: &kopiacmd.CommandArgs{
					RepoPassword:   "",
					ConfigFilePath: configFile,
					LogDirectory:   logDirectory,
				},
				Tags:                   tags,
				ProgressUpdateInterval: 0,
				Parallelism:            utils.GetEnvAsIntOrDefault(kankopia.DataStoreParallelUploadName, kankopia.DefaultDataStoreParallelUpload),
			})
		if err != nil {
			return nil, errors.Wrap(err, "Failed to construct snapshot create command")
		}
		stdout, stderr, err = kube.Exec(cli, namespace, pod.Name, pod.Spec.Containers[0].Name, cmd, nil)
		format.Log(pod.Name, pod.Spec.Containers[0].Name, stdout)
		format.Log(pod.Name, pod.Spec.Containers[0].Name, stderr)

		message := "Failed to create and upload backup"
		if err != nil {
			if strings.Contains(err.Error(), kerrors.ErrCodeOutOfMemoryStr) {
				message = message + ": " + kerrors.ErrOutOfMemoryStr
			}
			return nil, errors.Wrap(err, message)
		}
		// Parse logs and return snapshot IDs and stats
		snapInfo, err := kopiacmd.ParseSnapshotCreateOutput(stdout, stderr)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to parse snapshot create output")
		}

		var logSize, phySize int64
		if snapInfo.Stats != nil {
			stats := snapInfo.Stats
			logSize = stats.SizeHashedB + stats.SizeCachedB
			phySize = stats.SizeUploadedB
		}

		return map[string]interface{}{
			CopyVolumeDataOutputBackupID:     snapInfo.SnapshotID,
			CopyVolumeDataOutputBackupRoot:   mountPoint,
			CopyVolumeDataOutputBackupTag:    tags,
			CopyVolumeDataOutputBackupSize:   humanize.Bytes(uint64(logSize)),
			CopyVolumeDataOutputPhysicalSize: humanize.Bytes(uint64(phySize)),
		}, nil
	}
}
