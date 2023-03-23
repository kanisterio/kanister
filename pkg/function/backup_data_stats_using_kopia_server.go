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
	"github.com/dustin/go-humanize"
	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/format"
	kankopia "github.com/kanisterio/kanister/pkg/kopia"
	kopiacmd "github.com/kanisterio/kanister/pkg/kopia/command"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// BackupDataStatsUsingKopiaServerFuncName gives the name of the stats function
	BackupDataStatsUsingKopiaServerFuncName  = "BackupDataStatsUsingKopiaServer"
	backupDataUsingKopiaServerStatsJobPrefix = "backup-data-stats-using-kopia-server-"
)

type BackupDataStatsUsingKopiaServer struct{}

var _ kanister.Func = (*BackupDataStatsUsingKopiaServer)(nil)

func init() {
	_ = kanister.Register(&BackupDataStatsUsingKopiaServer{})
}

func (*BackupDataStatsUsingKopiaServer) Name() string {
	return BackupDataStatsUsingKopiaServerFuncName
}

func (*BackupDataStatsUsingKopiaServer) RequiredArgs() []string {
	return []string{
		BackupDataStatsNamespaceArg,
	}
}

func (*BackupDataStatsUsingKopiaServer) Arguments() []string {
	return []string{
		BackupDataStatsNamespaceArg,
		DescribeBackupsPodOverrideArg,
	}
}

func (*BackupDataStatsUsingKopiaServer) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var namespace string
	var err error
	if err = Arg(args, BackupDataStatsNamespaceArg, &namespace); err != nil {
		return nil, err
	}

	podOverride, err := GetPodSpecOverride(tp, args, DescribeBackupsPodOverrideArg)
	if err != nil {
		return nil, err
	}

	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}

	return backupDataStatsUsingKopiaServer(
		ctx,
		cli,
		tp,
		namespace,
		backupDataUsingKopiaServerStatsJobPrefix,
		podOverride,
	)

}

func backupDataStatsUsingKopiaServer(ctx context.Context, cli kubernetes.Interface, tp param.TemplateParams, namespace, jobPrefix string, podOverride crv1alpha1.JSONMap) (map[string]interface{}, error) {
	userPassphrase, cert, err := userCredentialsAndServerTLS(&tp)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to fetch User Credentials/Certificate Data from Template Params")
	}

	fingerprint, err := kankopia.ExtractFingerprintFromCertificateJSON(cert)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to fetch Kopia API Server Certificate Secret Data from Certificate")
	}

	hostname, userAccessPassphrase, err := hostNameAndUserPassPhraseFromRepoServer(userPassphrase)

	options := &kube.PodOptions{
		Namespace:    namespace,
		GenerateName: jobPrefix,
		Image:        getKanisterToolsImage(),
		Command:      []string{"sh", "-c", "tail -f /dev/null"},
		PodOverride:  podOverride,
	}
	pr := kube.NewPodRunner(cli, options)
	podFunc := backupDataStatsUsingKopiaServerPodFunc(
		cli,
		hostname,
		namespace,
		tp.RepositoryServer.Address,
		fingerprint,
		tp.RepositoryServer.Username,
		userAccessPassphrase,
	)
	return pr.Run(ctx, podFunc)
}

func backupDataStatsUsingKopiaServerPodFunc(
	cli kubernetes.Interface,
	hostname,
	namespace,
	serverAddress,
	fingerprint,
	username,
	userPassphrase string,
) func(ctx context.Context, pod *corev1.Pod) (map[string]any, error) {
	return func(ctx context.Context, pod *corev1.Pod) (map[string]any, error) {
		if err := kube.WaitForPodReady(ctx, cli, pod.Namespace, pod.Name); err != nil {
			return nil, errors.Wrap(err, "Failed while waiting for Pod: "+pod.Name+" to be ready")
		}

		contentCacheMB, metadataCacheMB := kopiacmd.GetCacheSizeSettingsForSnapshot()
		configFile, logDirectory := kankopia.CustomConfigFileAndLogDirectory(hostname)

		cmd := kopiacmd.RepositoryConnectServerCommand(
			kopiacmd.RepositoryServerCommandArgs{
				UserPassword:    userPassphrase,
				ConfigFilePath:  configFile,
				LogDirectory:    logDirectory,
				CacheDirectory:  kopiacmd.DefaultCacheDirectory,
				Hostname:        hostname,
				ServerURL:       serverAddress,
				Fingerprint:     fingerprint,
				Username:        username,
				ContentCacheMB:  contentCacheMB,
				MetadataCacheMB: metadataCacheMB,
			})
		stdout, stderr, err := kube.Exec(cli, namespace, pod.Name, pod.Spec.Containers[0].Name, cmd, nil)
		format.Log(pod.Name, pod.Spec.Containers[0].Name, stdout)
		format.Log(pod.Name, pod.Spec.Containers[0].Name, stderr)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to connect to Kopia API server")
		}

		cmd = kopiacmd.BlobStats(
			kopiacmd.BlobStatsCommandArgs{
				CommandArgs: &kopiacmd.CommandArgs{
					RepoPassword:   "",
					ConfigFilePath: configFile,
					LogDirectory:   logDirectory,
				},
			})
		stdout, stderr, err = kube.Exec(cli, namespace, pod.Name, pod.Spec.Containers[0].Name, cmd, nil)
		format.Log(pod.Name, pod.Spec.Containers[0].Name, stdout)
		format.Log(pod.Name, pod.Spec.Containers[0].Name, stderr)
		if err != nil {
			return nil, errors.Wrap(err, "Blob stats execution failed")
		}
		phySizeSum, _, err := kopiacmd.RepoSizeStatsFromBlobStatsRaw(stdout)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to get backup stats from 'kopia blob list' stats")
		}

		cmd = kopiacmd.SnapListAll(
			kopiacmd.SnapListAllCommandArgs{
				CommandArgs: &kopiacmd.CommandArgs{
					RepoPassword:   "",
					ConfigFilePath: configFile,
					LogDirectory:   logDirectory,
				},
			})
		stdout, stderr, err = kube.Exec(cli, namespace, pod.Name, pod.Spec.Containers[0].Name, cmd, nil)
		format.Log(pod.Name, pod.Spec.Containers[0].Name, stdout)
		format.Log(pod.Name, pod.Spec.Containers[0].Name, stderr)
		if err != nil {
			return nil, errors.Wrap(err, "Snapshot list execution failed")
		}
		logSizeSum, _, err := kopiacmd.SnapSizeStatsFromSnapListAll(stdout)

		output := map[string]any{
			BackupDataOutputBackupSize:         humanize.Bytes(uint64(logSizeSum)),
			BackupDataOutputBackupPhysicalSize: humanize.Bytes(uint64(phySizeSum)),
		}
		return output, nil
	}
}
