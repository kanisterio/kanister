/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package function

import (
	"context"
	"io"
	"time"

	"github.com/kanisterio/errkit"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/field"
	kopiacmd "github.com/kanisterio/kanister/pkg/kopia/command"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/utils"
	api "github.com/kastenhq/datamover/api/v1alpha1"
	"github.com/kastenhq/datamover/client"
)

// FIXME: arg to mount PVC as read-only
const (
	CopyVolumeDataDMFuncName           = "CopyVolumeDataDM"
	CopyVolumeDataDMArgNamespace       = "namespace"
	CopyVolumeDataDMArgImage           = "image"
	CopyVolumeDataDMArgVolume          = "volume" // TODO: PVC???
	CopyVolumeDataDMArgDatamoverServer = "datamoverServer"
	CopyVolumeDataDMArgDataPath        = "dataPath"     // TODO: dataPathPrefix???
	CopyVolumeDataDMArgTag             = "tag"          // Backup tag // FIXME: make optional?
	CopyVolumeDataDMArgClientSecret    = "clientSecret" // TODO: clientSecretVolume???
	CopyVolumeDataDMArgConfig          = "config"
	CopyVolumeDataDMArgSecrets         = "secrets"
	CopyVolumeDataDMArgEnv             = "env"
	CopyVolumeDataDMArgPodOptions      = "podOptions"
)

type CopyVolumeDataDM struct {
	Namespace          string
	Image              string
	Volume             string // PVC??
	DataMoverServerRef DataMoverServerRef
	DataPath           string
	Tag                string
	ClientSecret       string
	Secrets            []string
	ConfigMap          *string
	Env                []corev1.EnvVar
	PodOptions         api.PodOptions
	progressPercent    string
}

type DataMoverServerRef struct {
	Namespace string
	Name      string
}

func init() {
	_ = kanister.Register(&CopyVolumeDataDM{})
}

var _ kanister.Func = (*CopyVolumeDataDM)(nil)

// NOTE: since PVCs and secrets are namespaced, we need to have client secret in
// the same namespace as the PVC (and consequently the one for the pod)
// Client-Server secret association is still an open question, but it would be
// better to keep only relevant client secret in app namespace, which makes
// creation of server secrets FROM client secrets more attractive than other way around

func (cvd *CopyVolumeDataDM) Name() string {
	return CopyVolumeDataDMFuncName
}

func (cvd *CopyVolumeDataDM) RequiredArgs() []string {
	return []string{
		CopyVolumeDataDMArgNamespace,
		CopyVolumeDataDMArgImage,
		CopyVolumeDataDMArgVolume,
		CopyVolumeDataDMArgDatamoverServer,
		CopyVolumeDataDMArgDataPath,
		CopyVolumeDataDMArgClientSecret,
	}
}

func (cvd *CopyVolumeDataDM) Arguments() []string {
	return append(cvd.RequiredArgs(), []string{
		CopyVolumeDataDMArgConfig,
		CopyVolumeDataDMArgSecrets,
		CopyVolumeDataDMArgEnv,
		CopyVolumeDataDMArgPodOptions,
	}...)
}

func (cvd *CopyVolumeDataDM) Validate(args map[string]any) error {
	if err := utils.CheckSupportedArgs(cvd.Arguments(), args); err != nil {
		return err
	}

	return utils.CheckRequiredArgs(cvd.RequiredArgs(), args)
}

func (cvd *CopyVolumeDataDM) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {

	var err error
	if err = Arg(args, CopyVolumeDataDMArgNamespace, &cvd.Namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, CopyVolumeDataDMArgImage, &cvd.Image); err != nil {
		return nil, err
	}
	if err = Arg(args, CopyVolumeDataDMArgVolume, &cvd.Volume); err != nil {
		return nil, err
	}
	if err = Arg(args, CopyVolumeDataDMArgDataPath, &cvd.DataPath); err != nil {
		return nil, err
	}

	if err = OptArg(args, CopyVolumeDataDMArgTag, &cvd.Tag, ""); err != nil {
		return nil, err
	}

	// TODO: we can validate that this secret is in datamover clients secret if we have access to datamover server secrets
	var actionClientSecretName string
	if err = Arg(args, CopyVolumeDataDMArgClientSecret, &actionClientSecretName); err != nil {
		return nil, err
	}
	clientSecretSpec, ok := tp.Secrets[actionClientSecretName]
	if !ok {
		return nil, errkit.New("Client secret not found in the actionset:", "secretName", actionClientSecretName)
	}
	if clientSecretSpec.Namespace != cvd.Namespace {
		return nil, errkit.New("Client secret in the actionset is in the wrong namespace:", "secretName", actionClientSecretName, "secretNamespace", clientSecretSpec.Namespace, "namespace", cvd.Namespace)
	}
	cvd.ClientSecret = clientSecretSpec.Name

	var configmap string
	if err = OptArg(args, CopyVolumeDataDMArgConfig, &configmap, ""); err != nil {
		return nil, err
	}
	if configmap != "" {
		cvd.ConfigMap = &configmap
	}

	var actionSecrets []string
	if err = OptArg(args, CopyVolumeDataDMArgSecrets, &actionSecrets, []string{}); err != nil {
		return nil, err
	}

	secretNames := []string{}
	for _, actionSecret := range actionSecrets {
		secretSpec, ok := tp.Secrets[actionSecret]
		if ok {
			if secretSpec.Namespace == cvd.Namespace {
				secretNames = append(secretNames, secretSpec.Name)
			} else {
				log.Info().Print("Secret reference from different namespace. Ignoring", field.M{"secretName": secretSpec.Name, "secretNamespace": secretSpec.Namespace})
			}
		}
	}
	cvd.Secrets = secretNames

	var podOptions *api.PodOptions
	if err = OptArg(args, CopyVolumeDataDMArgPodOptions, &podOptions, nil); err != nil {
		return nil, err
	}
	if podOptions == nil {
		cvd.PodOptions = api.PodOptions{}
	} else {
		cvd.PodOptions = *podOptions
	}

	var env map[string]string
	if err = OptArg(args, CopyVolumeDataDMArgEnv, &env, map[string]string{}); err != nil {
		return nil, err
	}
	cvd.Env = []corev1.EnvVar{}
	for k, v := range env {
		cvd.Env = append(cvd.Env, corev1.EnvVar{Name: k, Value: v})
	}

	var serverRef DataMoverServerRef
	if err = Arg(args, CopyVolumeDataDMArgDatamoverServer, &serverRef); err != nil {
		return nil, err
	}

	cvd.DataMoverServerRef = serverRef

	return cvd.RunPod(ctx)
}

func (cvd *CopyVolumeDataDM) RunPod(ctx context.Context) (map[string]interface{}, error) {
	cvd.progressPercent = progress.StartedPercent
	defer func() { cvd.progressPercent = progress.CompletedPercent }()

	cli, err := kube.NewClient()
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to create Kubernetes client")
	}

	dynCli, err := kube.NewDynamicClient()
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to create dynamic Kubernetes client")
	}
	// FIXME: set owner reference for created pod (actionset)
	pod, err := client.CreateClientPod(ctx, cli, dynCli, client.CreateClientArgs{
		// FIXME: read-only volume mount
		Operation:       client.FileSystemBackupOperation{Path: cvd.DataPath, Tag: cvd.Tag, PVC: cvd.Volume},
		Namespace:       cvd.Namespace,
		Image:           cvd.Image,
		ServerNamespace: cvd.DataMoverServerRef.Namespace,
		ServerName:      cvd.DataMoverServerRef.Name,
		ConfigMap:       cvd.ConfigMap,
		Secrets:         cvd.Secrets,
		Env:             cvd.Env,
		PodOptions:      cvd.PodOptions,
		CredentialsConfig: client.ClientCredentialsConfig{
			CredentialsMode: client.ClientCredentialsSecret,
			SecretName:      &cvd.ClientSecret,
		},
	})

	if err != nil {
		return nil, errkit.Wrap(err, "Unable to create pod")
	}

	pc, err := cvd.runPod(ctx, cli, pod)
	if err != nil {
		return nil, errkit.Wrap(err, "Pod run error")
	}

	podOutput, err := cvd.getPodLogs(ctx, pc)
	if err != nil {
		return nil, errkit.Wrap(err, "Cannot get pod logs")
	}

	log.Info().Print("Pod output", field.M{"PodOutput": podOutput})

	snapInfo, err := kopiacmd.ParseSnapshotCreateOutput(podOutput, "")
	if err != nil {
		return nil, errkit.Wrap(err, "Cannot parse kopia snapshot create output")
	}

	// FIXME: this needs to be checked. Some inconsistency in stats
	var logSize, phySize, fileCount int64
	if snapInfo.Stats != nil {
		stats := snapInfo.Stats
		logSize = stats.SizeHashedB + stats.SizeCachedB
		phySize = stats.SizeUploadedB
		fileCount = stats.FilesHashed + stats.FilesCached
	}

	output := map[string]any{
		CopyVolumeDataOutputBackupID:        snapInfo.SnapshotID,
		CopyVolumeDataOutputBackupSize:      logSize,
		CopyVolumeDataOutputPhysicalSize:    phySize,
		CopyVolumeDataOutputBackupFileCount: fileCount,
	}
	return output, nil
}

func (cvd *CopyVolumeDataDM) runPod(ctx context.Context, cli kubernetes.Interface, pod *corev1.Pod) (kube.PodController, error) {
	pc, err := kube.NewPodControllerForExistingPod(cli, pod)
	if err != nil {
		return nil, err
	}

	ctx = field.Context(ctx, consts.PodNameKey, pod.Name)
	ctx = field.Context(ctx, consts.ContainerNameKey, pod.Spec.Containers[0].Name)
	go func() {
		<-ctx.Done()
		err := pc.StopPod(context.Background(), kube.PodControllerInfiniteStopTime, int64(0))
		if err != nil {
			log.WithError(err).Print("Failed to delete pod", field.M{"PodName": pod.Name})
		}
	}()

	if err := pc.WaitForPodReady(ctx); err != nil {
		return nil, errkit.Wrap(err, "Failed while waiting for Pod to be ready", "pod", pc.PodName())
	}

	// FIXME: update progress percent

	// Wait for pod completion
	if err := pc.WaitForPodCompletion(ctx); err != nil {
		return nil, errkit.Wrap(err, "Failed while waiting for Pod to complete", "pod", pc.PodName())
	}
	return pc, nil
}

func (cvd *CopyVolumeDataDM) getPodLogs(ctx context.Context, pc kube.PodController) (string, error) {
	ctx = field.Context(ctx, consts.LogKindKey, consts.LogKindDatapath)
	// Fetch logs from the pod
	r, err := pc.StreamPodLogs(ctx)
	if err != nil {
		return "", errkit.Wrap(err, "Failed to fetch logs from the pod")
	}
	// FIXME: k8s logs stdout and stderro together. Do we need to separate them here?
	stdout, err := io.ReadAll(r)
	if err != nil {
		return "", errkit.Wrap(err, "Failed to read logs stream from the pod")
	}
	return string(stdout), nil
}

func (cvd *CopyVolumeDataDM) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    cvd.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}
