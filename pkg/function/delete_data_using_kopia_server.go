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
	"bytes"
	"context"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/ephemeral"
	"github.com/kanisterio/kanister/pkg/format"
	kankopia "github.com/kanisterio/kanister/pkg/kopia"
	kopiacmd "github.com/kanisterio/kanister/pkg/kopia/command"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/utils"
)

const (
	DeleteDataUsingKopiaServerFuncName = "DeleteDataUsingKopiaServer"
)

type deleteDataUsingKopiaServerFunc struct {
	progressPercent string
}

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
		KopiaRepositoryServerUserHostname,
		PodAnnotationsArg,
		PodLabelsArg,
	}
}

func (d *deleteDataUsingKopiaServerFunc) Validate(args map[string]any) error {
	if err := ValidatePodLabelsAndAnnotations(d.Name(), args); err != nil {
		return err
	}

	if err := utils.CheckSupportedArgs(d.Arguments(), args); err != nil {
		return err
	}

	return utils.CheckRequiredArgs(d.RequiredArgs(), args)
}

func (d *deleteDataUsingKopiaServerFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]any) (map[string]any, error) {
	// Set progress percent
	d.progressPercent = progress.StartedPercent
	defer func() { d.progressPercent = progress.CompletedPercent }()

	var (
		err           error
		image         string
		namespace     string
		snapID        string
		userHostname  string
		bpAnnotations map[string]string
		bpLabels      map[string]string
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
	if err = OptArg(args, KopiaRepositoryServerUserHostname, &userHostname, ""); err != nil {
		return nil, err
	}
	if err = OptArg(args, PodAnnotationsArg, &bpAnnotations, nil); err != nil {
		return nil, err
	}
	if err = OptArg(args, PodLabelsArg, &bpLabels, nil); err != nil {
		return nil, err
	}

	annotations := bpAnnotations
	labels := bpLabels
	if tp.PodAnnotations != nil {
		// merge the actionset annotations with blueprint annotations
		var actionSetAnn ActionSetAnnotations = tp.PodAnnotations
		annotations = actionSetAnn.MergeBPAnnotations(bpAnnotations)
	}

	if tp.PodLabels != nil {
		// merge the actionset labels with blueprint labels
		var actionSetLabels ActionSetLabels = tp.PodLabels
		labels = actionSetLabels.MergeBPLabels(bpLabels)
	}

	userPassphrase, cert, err := userCredentialsAndServerTLS(&tp)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to fetch User Credentials/Certificate Data from Template Params")
	}

	fingerprint, err := kankopia.ExtractFingerprintFromCertificateJSON(cert)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to fetch Kopia API Server Certificate Secret Data from Certificate")
	}

	hostname, userAccessPassphrase, err := hostNameAndUserPassPhraseFromRepoServer(userPassphrase, userHostname)
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
		tp.RepositoryServer.Username,
		userAccessPassphrase,
		annotations,
		labels,
	)
}

func (d *deleteDataUsingKopiaServerFunc) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    d.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
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
	annotations,
	labels map[string]string,
) (map[string]any, error) {
	options := &kube.PodOptions{
		Namespace:    namespace,
		GenerateName: jobPrefix,
		Image:        image,
		Command:      []string{"bash", "-c", "tail -f /dev/null"},
		Annotations:  annotations,
		Labels:       labels,
	}

	// Apply the registered ephemeral pod changes.
	ephemeral.PodOptions.Apply(options)

	pr := kube.NewPodRunner(cli, options)
	podFunc := deleteDataFromServerPodFunc(
		hostname,
		serverAddress,
		fingerprint,
		snapID,
		username,
		userPassphrase,
	)
	return pr.Run(ctx, podFunc)
}

func deleteDataFromServerPodFunc(
	hostname,
	serverAddress,
	fingerprint,
	snapID,
	username,
	userPassphrase string,
) func(ctx context.Context, pc kube.PodController) (map[string]any, error) {
	return func(ctx context.Context, pc kube.PodController) (map[string]any, error) {
		pod := pc.Pod()

		// Wait for pod to reach running state
		if err := pc.WaitForPodReady(ctx); err != nil {
			return nil, errors.Wrapf(err, "Failed while waiting for Pod %s to be ready", pod.Name)
		}

		contentCacheMB, metadataCacheMB := kopiacmd.GetCacheSizeSettingsForSnapshot()
		configFile, logDirectory := kankopia.CustomConfigFileAndLogDirectory(hostname)
		cmd := kopiacmd.RepositoryConnectServerCommand(
			kopiacmd.RepositoryServerCommandArgs{
				UserPassword:   userPassphrase,
				ConfigFilePath: configFile,
				LogDirectory:   logDirectory,
				CacheDirectory: kopiacmd.DefaultCacheDirectory,
				Hostname:       hostname,
				ServerURL:      serverAddress,
				Fingerprint:    fingerprint,
				Username:       username,
				CacheArgs: kopiacmd.CacheArgs{
					ContentCacheLimitMB:  contentCacheMB,
					MetadataCacheLimitMB: metadataCacheMB,
				},
			})

		commandExecutor, err := pc.GetCommandExecutor()
		if err != nil {
			return nil, errors.Wrap(err, "Unable to get pod command executor")
		}

		var stdout, stderr bytes.Buffer
		err = commandExecutor.Exec(ctx, cmd, nil, &stdout, &stderr)
		format.LogWithCtx(ctx, pod.Name, pod.Spec.Containers[0].Name, stdout.String())
		format.LogWithCtx(ctx, pod.Name, pod.Spec.Containers[0].Name, stderr.String())
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
		stdout.Reset()
		stderr.Reset()
		err = commandExecutor.Exec(ctx, cmd, nil, &stdout, &stderr)
		format.LogWithCtx(ctx, pod.Name, pod.Spec.Containers[0].Name, stdout.String())
		format.LogWithCtx(ctx, pod.Name, pod.Spec.Containers[0].Name, stderr.String())
		return nil, errors.Wrap(err, "Failed to delete backup from Kopia API server")
	}
}
