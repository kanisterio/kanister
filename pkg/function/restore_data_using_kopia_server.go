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

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/format"
	kankopia "github.com/kanisterio/kanister/pkg/kopia"
	kopiacmd "github.com/kanisterio/kanister/pkg/kopia/command"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
)

const (
	RestoreDataUsingKopiaServerFuncName = "RestoreDataUsingKopiaServer"
	// SparseRestoreOption is the key for specifiying whether to do a sparse restore
	SparseRestoreOption = "sparseRestore"
)

type restoreDataUsingKopiaServerFunc struct{}

func init() {
	_ = kanister.Register(&restoreDataUsingKopiaServerFunc{})
}

var _ kanister.Func = (*restoreDataUsingKopiaServerFunc)(nil)

func (*restoreDataUsingKopiaServerFunc) Name() string {
	return RestoreDataUsingKopiaServerFuncName
}

func (*restoreDataUsingKopiaServerFunc) RequiredArgs() []string {
	return []string{
		RestoreDataBackupIdentifierArg,
		RestoreDataNamespaceArg,
		RestoreDataRestorePathArg,
		RestoreDataImageArg,
	}
}

func (*restoreDataUsingKopiaServerFunc) Arguments() []string {
	return []string{
		RestoreDataBackupIdentifierArg,
		RestoreDataNamespaceArg,
		RestoreDataRestorePathArg,
		RestoreDataPodArg,
		RestoreDataVolsArg,
		RestoreDataPodOverrideArg,
		RestoreDataImageArg,
	}
}

func (*restoreDataUsingKopiaServerFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]any) (map[string]any, error) {
	var (
		err         error
		image       string
		namespace   string
		restorePath string
		snapID      string
	)
	if err = Arg(args, RestoreDataBackupIdentifierArg, &snapID); err != nil {
		return nil, err
	}
	if err = Arg(args, RestoreDataNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, RestoreDataRestorePathArg, &restorePath); err != nil {
		return nil, err
	}
	if err = Arg(args, RestoreDataImageArg, &image); err != nil {
		return nil, err
	}

	userPassphrase, cert, err := userCredentialsAndServerTLS(&tp)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to fetch User Credentials/Certificate Data from Template Params")
	}

	fingerprint, err := kankopia.ExtractFingerprintFromCertificateJSON(cert)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to fetch Kopia API Server Certificate Secret Data from Certificate")
	}

	// Validate and get optional arguments
	pod, vols, podOverride, err := validateAndGetOptArgsForRestore(tp, args)
	if err != nil {
		return nil, err
	}

	if len(vols) == 0 {
		vols, err = FetchPodVolumes(pod, tp)
		if err != nil {
			return nil, err
		}
	}

	username := tp.RepositoryServer.Username
	hostname, userAccessPassphrase, err := hostNameAndUserPassPhraseFromRepoServer(userPassphrase)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get hostname/user passphrase from Options")
	}

	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create Kubernetes client")
	}

	_, sparseRestore := tp.Options[SparseRestoreOption]

	return restoreDataFromServer(
		ctx,
		cli,
		hostname,
		image,
		restoreDataJobPrefix,
		namespace,
		restorePath,
		tp.RepositoryServer.Address,
		fingerprint,
		snapID,
		username,
		userAccessPassphrase,
		sparseRestore,
		vols,
		podOverride,
	)
}

func restoreDataFromServer(
	ctx context.Context,
	cli kubernetes.Interface,
	hostname,
	image,
	jobPrefix,
	namespace,
	restorePath,
	serverAddress,
	fingerprint,
	snapID,
	username,
	userPassphrase string,
	sparseRestore bool,
	vols map[string]string,
	podOverride crv1alpha1.JSONMap,
) (map[string]any, error) {
	for pvc := range vols {
		if _, err := cli.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, pvc, metav1.GetOptions{}); err != nil {
			return nil, errors.Wrap(err, "Failed to retrieve PVC from namespace: "+namespace+" name: "+pvc)
		}
	}

	options := &kube.PodOptions{
		Namespace:    namespace,
		GenerateName: jobPrefix,
		Image:        image,
		Command:      []string{"bash", "-c", "tail -f /dev/null"},
		Volumes:      vols,
		PodOverride:  podOverride,
	}

	pr := kube.NewPodRunner(cli, options)
	podFunc := restoreDataFromServerPodFunc(
		cli,
		hostname,
		namespace,
		restorePath,
		serverAddress,
		fingerprint,
		snapID,
		username,
		userPassphrase,
		sparseRestore,
	)
	return pr.Run(ctx, podFunc)
}

func restoreDataFromServerPodFunc(
	cli kubernetes.Interface,
	hostname,
	namespace,
	restorePath,
	serverAddress,
	fingerprint,
	snapID,
	username,
	userPassphrase string,
	sparseRestore bool,
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

		cmd = kopiacmd.SnapshotRestore(
			kopiacmd.SnapshotRestoreCommandArgs{
				CommandArgs: &kopiacmd.CommandArgs{
					RepoPassword:   "",
					ConfigFilePath: configFile,
					LogDirectory:   logDirectory,
				},
				SnapID:                 snapID,
				TargetPath:             restorePath,
				SparseRestore:          sparseRestore,
				IgnorePermissionErrors: true,
			})
		stdout, stderr, err = kube.Exec(cli, namespace, pod.Name, pod.Spec.Containers[0].Name, cmd, nil)
		format.Log(pod.Name, pod.Spec.Containers[0].Name, stdout)
		format.Log(pod.Name, pod.Spec.Containers[0].Name, stderr)
		return nil, errors.Wrap(err, "Failed to restore backup from Kopia API server")
	}
}

func validateAndGetOptArgsForRestore(tp param.TemplateParams, args map[string]any) (pod string, vols map[string]string, podOverride crv1alpha1.JSONMap, err error) {
	if err = OptArg(args, RestoreDataPodArg, &pod, ""); err != nil {
		return pod, vols, podOverride, err
	}
	if err = OptArg(args, RestoreDataVolsArg, &vols, nil); err != nil {
		return pod, vols, podOverride, err
	}
	if (pod != "") && (len(vols) > 0) {
		return pod, vols, podOverride, errors.New(fmt.Sprintf("Exactly one of the %s or %s arguments are required, but both are provided", RestoreDataPodArg, RestoreDataVolsArg))
	}
	podOverride, err = GetPodSpecOverride(tp, args, RestoreDataPodOverrideArg)
	if err != nil {
		return pod, vols, podOverride, errors.Wrap(err, "Failed to get Pod Override Specs")
	}
	return pod, vols, podOverride, nil
}
