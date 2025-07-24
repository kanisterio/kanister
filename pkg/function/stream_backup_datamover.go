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
	"github.com/kastenhq/datamover/client"
)

const (
	StreamBackupDMFuncName            = "StreamBackupDM"
	StreamBackupDMArgNamespace        = "namespace"
	StreamBackupDMArgImage            = "image"
	StreamBackupDMArgDatamoverServer  = "datamoverServer"
	StreamBackupDMArgGenerator        = "streamGenerator"
	StreamBackupDMArgBackupObjectName = "backupObjectName"
	StreamBackupDMArgTag              = "tag" // Backup tag
	StreamBackupDMArgInitImage        = "initImage"
	StreamBackupDMArgClientSecret     = "clientSecret" // TODO: clientSecretVolume???
	StreamBackupDMArgConfig           = "config"
	StreamBackupDMArgSecrets          = "secrets"
)

type StreamBackupDM struct {
	Namespace          string
	Image              string
	DataMoverServerRef DataMoverServerRef
	StreamGenerator    corev1.Container
	BackupObjectName   string
	Tag                string
	InitImage          string
	ClientSecret       string
	Secrets            []string
	ConfigMap          *string
	progressPercent    string
}

func init() {
	_ = kanister.Register(&StreamBackupDM{})
}

var _ kanister.Func = (*StreamBackupDM)(nil)

// NOTE: since PVCs and secrets are namespaced, we need to have client secret in
// the same namespace as the PVC (and consequently the one for the pod)
// Client-Server secret association is still an open question, but it would be
// better to keep only relevant client secret in app namespace, which makes
// creation of server secrets FROM client secrets more attractive than other way around

func (streamBackup *StreamBackupDM) Name() string {
	return StreamBackupDMFuncName
}

func (streamBackup *StreamBackupDM) RequiredArgs() []string {
	return []string{
		StreamBackupDMArgNamespace,
		StreamBackupDMArgImage,
		StreamBackupDMArgDatamoverServer,
		StreamBackupDMArgGenerator,
		StreamBackupDMArgClientSecret,
		// TODO: implementation specific secrets
		// TLS fingerprint secret
	}
}

func (streamBackup *StreamBackupDM) Arguments() []string {
	return append(streamBackup.RequiredArgs(), []string{
		StreamBackupDMArgTag,
		StreamBackupDMArgBackupObjectName,
		StreamBackupDMArgInitImage,
		StreamBackupDMArgConfig,
		StreamBackupDMArgSecrets,
	}...)
}

func (streamBackup *StreamBackupDM) Validate(args map[string]any) error {
	if err := utils.CheckSupportedArgs(streamBackup.Arguments(), args); err != nil {
		return err
	}

	return utils.CheckRequiredArgs(streamBackup.RequiredArgs(), args)
	// TODO: validate that generator is a container and datamover session
}

func (streamBackup *StreamBackupDM) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {

	streamBackup.progressPercent = progress.StartedPercent
	defer func() { streamBackup.progressPercent = progress.CompletedPercent }()

	var err error
	if err = Arg(args, StreamBackupDMArgNamespace, &streamBackup.Namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, StreamBackupDMArgImage, &streamBackup.Image); err != nil {
		return nil, err
	}

	var serverRef DataMoverServerRef
	if err = Arg(args, StreamBackupDMArgDatamoverServer, &serverRef); err != nil {
		return nil, err
	}
	streamBackup.DataMoverServerRef = serverRef

	var generatorContainer corev1.Container
	if err = Arg(args, StreamBackupDMArgGenerator, &generatorContainer); err != nil {
		return nil, err
	}
	streamBackup.StreamGenerator = generatorContainer

	// TODO: we can validate that this secret is in datamover clients secret if we have access to datamover server secrets
	var actionClientSecretName string
	if err = Arg(args, StreamBackupDMArgClientSecret, &actionClientSecretName); err != nil {
		return nil, err
	}

	if err = OptArg(args, StreamBackupDMArgTag, &streamBackup.Tag, ""); err != nil {
		return nil, err
	}

	clientSecretSpec, ok := tp.Secrets[actionClientSecretName]
	if !ok {
		return nil, errkit.New("Client secret not found in the actionset:", "secretName", actionClientSecretName)
	}
	if clientSecretSpec.Namespace != streamBackup.Namespace {
		return nil, errkit.New("Client secret in the actionset is in the wrong namespace:", "secretName", actionClientSecretName, "secretNamespace", clientSecretSpec.Namespace, "namespace", streamBackup.Namespace)
	}
	streamBackup.ClientSecret = clientSecretSpec.Name

	if err = OptArg(args, StreamBackupDMArgBackupObjectName, &streamBackup.BackupObjectName, "data"); err != nil {
		return nil, err
	}

	if err = OptArg(args, StreamBackupDMArgInitImage, &streamBackup.InitImage, ""); err != nil {
		return nil, err
	}

	// FIXME: configmap from actionset
	var configmap string
	if err = OptArg(args, StreamBackupDMArgConfig, &configmap, ""); err != nil {
		return nil, err
	}
	if configmap != "" {
		streamBackup.ConfigMap = &configmap
	}

	var actionSecrets []string
	if err = OptArg(args, StreamBackupDMArgSecrets, &actionSecrets, []string{}); err != nil {
		return nil, err
	}

	secretNames := []string{}
	for _, actionSecret := range actionSecrets {
		secretSpec, ok := tp.Secrets[actionSecret]
		if ok {
			if secretSpec.Namespace == streamBackup.Namespace {
				secretNames = append(secretNames, secretSpec.Name)
			} else {
				log.Info().Print("Secret reference from different namespace. Ignoring", field.M{"secretName": secretSpec.Name, "secretNamespace": secretSpec.Namespace})
			}
		}
	}
	streamBackup.Secrets = secretNames

	return streamBackup.RunPod(ctx)
}

func (streamBackup *StreamBackupDM) RunPod(ctx context.Context) (map[string]interface{}, error) {
	cli, err := kube.NewClient()
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to create Kubernetes client")
	}

	dynCli, err := kube.NewDynamicClient()
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to create dynamic Kubernetes client")
	}

	pod, err := client.CreateClientPod(ctx, cli, dynCli, client.CreateClientArgs{
		Operation: client.StreamBackupOperation{
			Tag:              streamBackup.Tag,
			StreamGenerator:  streamBackup.StreamGenerator,
			BackupObjectName: streamBackup.BackupObjectName,
			InitImage:        streamBackup.InitImage},
		Namespace:        streamBackup.Namespace,
		Image:            streamBackup.Image,
		SessionNamespace: streamBackup.DataMoverServerRef.Namespace,
		SessionName:      streamBackup.DataMoverServerRef.Name,
		ConfigMap:        streamBackup.ConfigMap,
		Secrets:          streamBackup.Secrets,
		CredentialsConfig: client.ClientCredentialsSecret{
			SecretName: streamBackup.ClientSecret,
		},
	})

	if err != nil {
		return nil, errkit.Wrap(err, "Unable to create pod")
	}

	pc, err := streamBackup.runPod(ctx, cli, pod)
	if err != nil {
		return nil, errkit.Wrap(err, "Pod run error")
	}

	podOutput, err := streamBackup.getPodLogs(ctx, pc)
	if err != nil {
		return nil, errkit.Wrap(err, "Cannot get pod logs")
	}

	log.Info().Print("Pod output", field.M{"PodOutput": podOutput})

	snapInfo, err := kopiacmd.ParseSnapshotCreateOutput(podOutput, podOutput)
	if err != nil {
		return nil, errkit.Wrap(err, "Cannot parse kopia snapshot create output")
	}

	log.Info().Print("Snapshot info", field.M{"info": snapInfo})

	// FIXME: this needs to be checked. Some inconsistency in stats
	var logSize, phySize, fileCount int64
	if snapInfo.Stats != nil {
		stats := snapInfo.Stats
		logSize = stats.SizeHashedB + stats.SizeCachedB
		phySize = stats.SizeUploadedB
		fileCount = stats.FilesHashed + stats.FilesCached
	}

	output := map[string]any{
		CopyVolumeDataOutputRootID:          snapInfo.RootID,
		CopyVolumeDataOutputBackupID:        snapInfo.SnapshotID,
		CopyVolumeDataOutputBackupSize:      logSize,
		CopyVolumeDataOutputPhysicalSize:    phySize,
		CopyVolumeDataOutputBackupFileCount: fileCount,
		CopyVolumeDataOutputPodName:         pod.Name,
	}
	return output, nil
}

func (streamBackup *StreamBackupDM) runPod(ctx context.Context, cli kubernetes.Interface, pod *corev1.Pod) (kube.PodController, error) {
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

	// Wait for pod completion
	if err := pc.WaitForPodCompletion(ctx); err != nil {
		return nil, errkit.Wrap(err, "Failed while waiting for Pod to complete", "pod", pc.PodName())
	}
	return pc, nil
}

// FIXME: specify a container to get logs from (client.MainContainerName)
func (streamBackup *StreamBackupDM) getPodLogs(ctx context.Context, pc kube.PodController) (string, error) {
	ctx = field.Context(ctx, consts.LogKindKey, consts.LogKindDatapath)
	// Fetch logs from the pod
	r, err := pc.StreamPodLogs(ctx)
	if err != nil {
		return "", errkit.Wrap(err, "Failed to fetch logs from the pod")
	}
	// TODO: k8s logs stdout and stderro together. Do we need to separate them here?
	stdout, err := io.ReadAll(r)
	if err != nil {
		return "", errkit.Wrap(err, "Failed to read logs stream from the pod")
	}
	return string(stdout), nil
}

func (streamBackup *StreamBackupDM) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    streamBackup.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}
