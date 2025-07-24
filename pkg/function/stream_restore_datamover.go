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
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/output"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/utils"
	"github.com/kastenhq/datamover/client"
)

const (
	StreamRestoreDMFuncName            = "StreamRestoreDM"
	StreamRestoreDMArgNamespace        = "namespace"
	StreamRestoreDMArgImage            = "image"
	StreamRestoreDMArgDatamoverServer  = "datamoverServer"
	StreamRestoreDMArgIngestor         = "streamIngestor"
	StreamRestoreDMArgBackupObjectName = "backupObjectName"
	StreamRestoreDMArgBackupId         = "backupId" // Backup tag // FIXME: make optional?
	StreamRestoreDMArgInitImage        = "initImage"
	StreamRestoreDMArgClientSecret     = "clientSecret" // TODO: clientSecretVolume???
	StreamRestoreDMArgConfig           = "config"
	StreamRestoreDMArgSecrets          = "secrets"
)

type StreamRestoreDM struct {
	Namespace          string
	Image              string
	DataMoverServerRef DataMoverServerRef
	StreamIngestor     corev1.Container
	BackupObjectName   string
	BackupId           string
	InitImage          string
	ClientSecret       string
	Secrets            []string
	ConfigMap          *string
	progressPercent    string
}

func init() {
	_ = kanister.Register(&StreamRestoreDM{})
}

var _ kanister.Func = (*StreamRestoreDM)(nil)

// NOTE: since PVCs and secrets are namespaced, we need to have client secret in
// the same namespace as the PVC (and consequently the one for the pod)
// Client-Server secret association is still an open question, but it would be
// better to keep only relevant client secret in app namespace, which makes
// creation of server secrets FROM client secrets more attractive than other way around

func (streamRestore *StreamRestoreDM) Name() string {
	return StreamRestoreDMFuncName
}

func (streamRestore *StreamRestoreDM) RequiredArgs() []string {
	return []string{
		StreamRestoreDMArgNamespace,
		StreamRestoreDMArgImage,
		StreamRestoreDMArgDatamoverServer,
		StreamRestoreDMArgIngestor,
		StreamRestoreDMArgBackupId,
		StreamRestoreDMArgClientSecret,
		// TODO: implementation specific secrets
		// TLS fingerprint secret
	}
}

func (streamRestore *StreamRestoreDM) Arguments() []string {
	return append(streamRestore.RequiredArgs(), []string{
		StreamRestoreDMArgBackupObjectName,
		StreamRestoreDMArgInitImage,
		StreamRestoreDMArgConfig,
		StreamRestoreDMArgSecrets,
	}...)
}

func (streamRestore *StreamRestoreDM) Validate(args map[string]any) error {
	if err := utils.CheckSupportedArgs(streamRestore.Arguments(), args); err != nil {
		return err
	}

	return utils.CheckRequiredArgs(streamRestore.RequiredArgs(), args)
	// TODO: validate that generator is a container and datamover session
}

func (streamRestore *StreamRestoreDM) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {

	streamRestore.progressPercent = progress.StartedPercent
	defer func() { streamRestore.progressPercent = progress.CompletedPercent }()

	var err error
	if err = Arg(args, StreamRestoreDMArgNamespace, &streamRestore.Namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, StreamRestoreDMArgImage, &streamRestore.Image); err != nil {
		return nil, err
	}

	var serverRef DataMoverServerRef
	if err = Arg(args, StreamRestoreDMArgDatamoverServer, &serverRef); err != nil {
		return nil, err
	}
	streamRestore.DataMoverServerRef = serverRef

	var generatorContainer corev1.Container
	if err = Arg(args, StreamRestoreDMArgIngestor, &generatorContainer); err != nil {
		return nil, err
	}
	streamRestore.StreamIngestor = generatorContainer

	// TODO: we can validate that this secret is in datamover clients secret if we have access to datamover server secrets
	var actionClientSecretName string
	if err = Arg(args, StreamRestoreDMArgClientSecret, &actionClientSecretName); err != nil {
		return nil, err
	}

	if err = OptArg(args, StreamRestoreDMArgBackupId, &streamRestore.BackupId, ""); err != nil {
		return nil, err
	}

	clientSecretSpec, ok := tp.Secrets[actionClientSecretName]
	if !ok {
		return nil, errkit.New("Client secret not found in the actionset:", "secretName", actionClientSecretName)
	}
	if clientSecretSpec.Namespace != streamRestore.Namespace {
		return nil, errkit.New("Client secret in the actionset is in the wrong namespace:", "secretName", actionClientSecretName, "secretNamespace", clientSecretSpec.Namespace, "namespace", streamRestore.Namespace)
	}
	streamRestore.ClientSecret = clientSecretSpec.Name

	if err = OptArg(args, StreamRestoreDMArgBackupObjectName, &streamRestore.BackupObjectName, "data"); err != nil {
		return nil, err
	}

	if err = OptArg(args, StreamRestoreDMArgInitImage, &streamRestore.InitImage, ""); err != nil {
		return nil, err
	}

	var configmap string
	if err = OptArg(args, StreamRestoreDMArgConfig, &configmap, ""); err != nil {
		return nil, err
	}
	if configmap != "" {
		streamRestore.ConfigMap = &configmap
	}

	var actionSecrets []string
	if err = OptArg(args, StreamRestoreDMArgSecrets, &actionSecrets, []string{}); err != nil {
		return nil, err
	}

	secretNames := []string{}
	for _, actionSecret := range actionSecrets {
		secretSpec, ok := tp.Secrets[actionSecret]
		if ok {
			if secretSpec.Namespace == streamRestore.Namespace {
				secretNames = append(secretNames, secretSpec.Name)
			} else {
				log.Info().Print("Secret reference from different namespace. Ignoring", field.M{"secretName": secretSpec.Name, "secretNamespace": secretSpec.Namespace})
			}
		}
	}
	streamRestore.Secrets = secretNames

	return streamRestore.RunPod(ctx)
}

func (streamRestore *StreamRestoreDM) RunPod(ctx context.Context) (map[string]interface{}, error) {
	cli, err := kube.NewClient()
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to create Kubernetes client")
	}

	dynCli, err := kube.NewDynamicClient()
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to create dynamic Kubernetes client")
	}

	pod, err := client.CreateClientPod(ctx, cli, dynCli, client.CreateClientArgs{
		// FIXME: support tags??
		Operation: client.StreamRestoreOperation{
			BackupID:         streamRestore.BackupId,
			BackupObjectName: streamRestore.BackupObjectName,
			InitImage:        streamRestore.InitImage,
			StreamIngestor:   streamRestore.StreamIngestor,
		},
		Namespace:        streamRestore.Namespace,
		Image:            streamRestore.Image,
		SessionNamespace: streamRestore.DataMoverServerRef.Namespace,
		SessionName:      streamRestore.DataMoverServerRef.Name,
		ConfigMap:        streamRestore.ConfigMap,
		Secrets:          streamRestore.Secrets,
		CredentialsConfig: client.ClientCredentialsSecret{
			SecretName: streamRestore.ClientSecret,
		},
	})

	if err != nil {
		return nil, errkit.Wrap(err, "Unable to create pod")
	}

	_, err = streamRestore.runPod(ctx, cli, pod)
	if err != nil {
		return nil, errkit.Wrap(err, "Pod run error")
	}

	podOutput, err := streamRestore.getPodLogs(ctx, cli, *pod)
	if err != nil {
		return nil, errkit.Wrap(err, "Cannot get pod logs")
	}

	out, err := output.LogAndParse(ctx, podOutput)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (streamRestore *StreamRestoreDM) runPod(ctx context.Context, cli kubernetes.Interface, pod *corev1.Pod) (kube.PodController, error) {
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

func (streamRestore *StreamRestoreDM) getPodLogs(ctx context.Context, cli kubernetes.Interface, pod corev1.Pod) (io.ReadCloser, error) {
	ctx = field.Context(ctx, consts.LogKindKey, consts.LogKindDatapath)
	// Fetch logs from the pod
	// FIXME default value for injestor container name??
	reader, err := kube.StreamPodLogs(ctx, cli, pod.Namespace, pod.Name, streamRestore.StreamIngestor.Name)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to fetch logs from the pod")
	}
	return reader, nil
}

func (streamRestore *StreamRestoreDM) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    streamRestore.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}
