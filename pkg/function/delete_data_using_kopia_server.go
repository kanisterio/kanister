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

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/format"
	kankopia "github.com/kanisterio/kanister/pkg/kopia"
	kopiacmd "github.com/kanisterio/kanister/pkg/kopia/command"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
)

const (
	DeleteDataUsingKopiaServerFuncName = "DeleteDataUsingKopiaServer"
)

type deleteDataUsingKopiaServerFunc struct{}

func init() {
	err := kanister.Register(&deleteDataUsingKopiaServerFunc{})
	if err != nil {
		return
	}
}

var _ kanister.Func = (*deleteDataUsingKopiaServerFunc)(nil)

func (*deleteDataUsingKopiaServerFunc) Name() string {
	return DeleteDataUsingKopiaServerFuncName
}

func (*deleteDataUsingKopiaServerFunc) RequiredArgs() []string {
	return []string{
		DeleteDataBackupIdentifierArg,
		DeleteDataNamespaceArg,
		RestoreDataImageArg,
	}
}

func (*deleteDataUsingKopiaServerFunc) Arguments() []string {
	return []string{
		DeleteDataBackupIdentifierArg,
		DeleteDataNamespaceArg,
		RestoreDataImageArg,
	}
}

func (*deleteDataUsingKopiaServerFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]any) (map[string]any, error) {
	var (
		err       error
		image     string
		namespace string
		snapID    string
	)
	if err = Arg(args, DeleteDataBackupIdentifierArg, &snapID); err != nil {
		return nil, err
	}
	if err = Arg(args, DeleteDataNamespaceArg, &namespace); err != nil {
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

	username := tp.RepositoryServer.Username
	hostname, userAccessPassphrase, err := hostNameAndUserPassPhraseFromRepoServer(userPassphrase)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get hostname/user passphrase from Options")
	}

	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create Kubernetes client")
	}

	return deleteDataFromServer(
		ctx,
		cli,
		hostname,
		image,
		deleteDataJobPrefix,
		namespace,
		tp.RepositoryServer.Address,
		fingerprint,
		snapID,
		username,
		userAccessPassphrase,
	)
}

func deleteDataFromServer(
	ctx context.Context,
	cli kubernetes.Interface,
	hostname,
	image,
	jobPrefix,
	namespace,
	serverAddress,
	fingerprint,
	snapID,
	username,
	userPassphrase string,
) (map[string]any, error) {
	options := &kube.PodOptions{
		Namespace:    namespace,
		GenerateName: jobPrefix,
		Image:        image,
		Command:      []string{"bash", "-c", "tail -f /dev/null"},
	}
	pr := kube.NewPodRunner(cli, options)
	podFunc := deleteDataFromServerPodFunc(
		cli,
		hostname,
		namespace,
		serverAddress,
		fingerprint,
		snapID,
		username,
		userPassphrase,
	)
	return pr.Run(ctx, podFunc)
}

func deleteDataFromServerPodFunc(
	cli kubernetes.Interface,
	hostname,
	namespace,
	serverAddress,
	fingerprint,
	snapID,
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
		stdout, stderr, err := kube.Exec(ctx, cli, namespace, pod.Name, pod.Spec.Containers[0].Name, cmd, nil)
		format.Log(pod.Name, pod.Spec.Containers[0].Name, stdout)
		format.Log(pod.Name, pod.Spec.Containers[0].Name, stderr)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to connect to Kopia Repository server")
		}

		cmd = kopiacmd.SnapshotDelete(
			kopiacmd.SnapshotDeleteCommandArgs{
				CommandArgs: &kopiacmd.CommandArgs{
					RepoPassword:   "",
					ConfigFilePath: configFile,
					LogDirectory:   logDirectory,
				},
				SnapID: snapID,
			})
		stdout, stderr, err = kube.Exec(ctx, cli, namespace, pod.Name, pod.Spec.Containers[0].Name, cmd, nil)
		format.Log(pod.Name, pod.Spec.Containers[0].Name, stdout)
		format.Log(pod.Name, pod.Spec.Containers[0].Name, stderr)
		return nil, errors.Wrap(err, "Failed to delete backup from Kopia API server")
	}
}
