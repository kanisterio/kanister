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
	"fmt"
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
	RestoreDataUsingKopiaServerFuncName = "RestoreDataUsingKopiaServer"
	// SparseRestoreOption is the key for specifying whether to do a sparse restore
	SparseRestoreOption = "sparseRestore"
)

type restoreDataUsingKopiaServerFunc struct {
	progressPercent string
}

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
		KopiaRepositoryServerUserHostname,
		PodAnnotationsArg,
		PodLabelsArg,
	}
}

func (r *restoreDataUsingKopiaServerFunc) Validate(args map[string]any) error {
	if err := ValidatePodLabelsAndAnnotations(r.Name(), args); err != nil {
		return err
	}

	if err := utils.CheckSupportedArgs(r.Arguments(), args); err != nil {
		return err
	}

	return utils.CheckRequiredArgs(r.RequiredArgs(), args)
}

func (r *restoreDataUsingKopiaServerFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]any) (map[string]any, error) {
	// Set progress percent
	r.progressPercent = progress.StartedPercent
	defer func() { r.progressPercent = progress.CompletedPercent }()

	var (
		err           error
		image         string
		namespace     string
		restorePath   string
		snapID        string
		userHostname  string
		bpAnnotations map[string]string
		bpLabels      map[string]string
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

	hostname, userAccessPassphrase, err := hostNameAndUserPassPhraseFromRepoServer(userPassphrase, userHostname)
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
		tp.RepositoryServer.Username,
		userAccessPassphrase,
		sparseRestore,
		vols,
		podOverride,
		annotations,
		labels,
	)
}

func (r *restoreDataUsingKopiaServerFunc) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    r.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
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
	annotations,
	labels map[string]string,
) (map[string]any, error) {
	validatedVols := make(map[string]kube.VolumeMountOptions)
	// Validate volumes
	for pvcName, mountPoint := range vols {
		pvc, err := cli.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, pvcName, metav1.GetOptions{})
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to retrieve PVC. Namespace %s, Name %s", namespace, pvcName)
		}

		validatedVols[pvcName] = kube.VolumeMountOptions{
			MountPath: mountPoint,
			ReadOnly:  kube.PVCContainsReadOnlyAccessMode(pvc),
		}
	}

	options := &kube.PodOptions{
		Namespace:    namespace,
		GenerateName: jobPrefix,
		Image:        image,
		Command:      []string{"bash", "-c", "tail -f /dev/null"},
		Volumes:      validatedVols,
		PodOverride:  podOverride,
		Annotations:  annotations,
		Labels:       labels,
	}

	// Apply the registered ephemeral pod changes.
	ephemeral.PodOptions.Apply(options)

	pr := kube.NewPodRunner(cli, options)
	podFunc := restoreDataFromServerPodFunc(
		hostname,
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
	hostname,
	restorePath,
	serverAddress,
	fingerprint,
	snapID,
	username,
	userPassphrase string,
	sparseRestore bool,
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
			return nil, errors.Wrapf(err, "Failed to connect to Kopia Repository server")
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
				Parallelism:            utils.GetEnvAsIntOrDefault(kankopia.DataStoreParallelDownloadName, kankopia.DefaultDataStoreParallelDownload),
			})

		stdout.Reset()
		stderr.Reset()
		err = commandExecutor.Exec(ctx, cmd, nil, &stdout, &stderr)
		format.LogWithCtx(ctx, pod.Name, pod.Spec.Containers[0].Name, stdout.String())
		format.LogWithCtx(ctx, pod.Name, pod.Spec.Containers[0].Name, stderr.String())

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
