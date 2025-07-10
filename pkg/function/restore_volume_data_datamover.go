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
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/utils"
	"github.com/kastenhq/datamover/client"
)

const (
	RestoreVolumeDataDMFuncName           = "RestoreVolumeDataDM"
	RestoreVolumeDataDMArgNamespace       = "namespace"
	RestoreVolumeDataDMArgImage           = "image"
	RestoreVolumeDataDMArgVolume          = "volume" // TODO: PVC???
	RestoreVolumeDataDMArgDatamoverServer = "datamoverServer"
	RestoreVolumeDataDMArgDataPath        = "dataPath"     // TODO: dataPathPrefix???
	RestoreVolumeDataDMArgBackupId        = "backupID"     // Backup id
	RestoreVolumeDataDMArgClientSecret    = "clientSecret" // TODO: clientSecretVolume???
	RestoreVolumeDataDMArgConfig          = "config"
	RestoreVolumeDataDMArgSecrets         = "secrets"
)

type RestoreVolumeDataDM struct {
	Namespace          string
	Image              string
	Volume             string // PVC??
	DataMoverServerRef DataMoverServerRef
	DataPath           string
	BackupId           string
	ClientSecret       string
	Secrets            []string
	ConfigMap          *string
	progressPercent    string
}

func init() {
	_ = kanister.Register(&RestoreVolumeDataDM{})
}

var _ kanister.Func = (*RestoreVolumeDataDM)(nil)

// NOTE: since PVCs and secrets are namespaced, we need to have client secret in
// the same namespace as the PVC (and consequently the one for the pod)
// Client-Server secret association is still an open question, but it would be
// better to keep only relevant client secret in app namespace, which makes
// creation of server secrets FROM client secrets more attractive than other way around

func (rvd *RestoreVolumeDataDM) Name() string {
	return RestoreVolumeDataDMFuncName
}

func (rvd *RestoreVolumeDataDM) RequiredArgs() []string {
	return []string{
		RestoreVolumeDataDMArgNamespace,
		RestoreVolumeDataDMArgImage,
		RestoreVolumeDataDMArgVolume,
		RestoreVolumeDataDMArgDatamoverServer,
		RestoreVolumeDataDMArgDataPath,
		RestoreVolumeDataDMArgBackupId,
		RestoreVolumeDataDMArgClientSecret,
		// TODO: implementation specific secrets
		// TLS fingerprint secret
	}
}

func (rvd *RestoreVolumeDataDM) Arguments() []string {
	return append(rvd.RequiredArgs(), []string{
		RestoreVolumeDataDMArgConfig,
		RestoreVolumeDataDMArgSecrets,
	}...)
}

func (rvd *RestoreVolumeDataDM) Validate(args map[string]any) error {
	if err := utils.CheckSupportedArgs(rvd.Arguments(), args); err != nil {
		return err
	}

	return utils.CheckRequiredArgs(rvd.RequiredArgs(), args)
}

func (rvd *RestoreVolumeDataDM) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {

	rvd.progressPercent = progress.StartedPercent
	defer func() { rvd.progressPercent = progress.CompletedPercent }()

	var err error
	if err = Arg(args, RestoreVolumeDataDMArgNamespace, &rvd.Namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, RestoreVolumeDataDMArgImage, &rvd.Image); err != nil {
		return nil, err
	}
	if err = Arg(args, RestoreVolumeDataDMArgVolume, &rvd.Volume); err != nil {
		return nil, err
	}
	if err = Arg(args, RestoreVolumeDataDMArgDataPath, &rvd.DataPath); err != nil {
		return nil, err
	}

	if err = OptArg(args, RestoreVolumeDataDMArgBackupId, &rvd.BackupId, ""); err != nil {
		return nil, err
	}

	// TODO: we can validate that this secret is in datamover clients secret if we have access to datamover server secrets
	var actionClientSecretName string
	if err = Arg(args, RestoreVolumeDataDMArgClientSecret, &actionClientSecretName); err != nil {
		return nil, err
	}

	clientSecretSpec, ok := tp.Secrets[actionClientSecretName]
	if !ok {
		return nil, errkit.New("Client secret not found in the actionset:", "secretName", actionClientSecretName)
	}
	if clientSecretSpec.Namespace != rvd.Namespace {
		return nil, errkit.New("Client secret in the actionset is in the wrong namespace:", "secretName", actionClientSecretName, "secretNamespace", clientSecretSpec.Namespace, "namespace", rvd.Namespace)
	}
	rvd.ClientSecret = clientSecretSpec.Name

	var configmap string
	if err = OptArg(args, RestoreVolumeDataDMArgConfig, &configmap, ""); err != nil {
		return nil, err
	}
	if configmap != "" {
		rvd.ConfigMap = &configmap
	}

	var actionSecrets []string
	if err = OptArg(args, RestoreVolumeDataDMArgSecrets, &actionSecrets, []string{}); err != nil {
		return nil, err
	}

	secretNames := []string{}
	for _, actionSecret := range actionSecrets {
		secretSpec, ok := tp.Secrets[actionSecret]
		if ok {
			if secretSpec.Namespace == rvd.Namespace {
				secretNames = append(secretNames, secretSpec.Name)
			} else {
				log.Info().Print("Secret reference from different namespace. Ignoring", field.M{"secretName": secretSpec.Name, "secretNamespace": secretSpec.Namespace})
			}
		}
	}
	rvd.Secrets = secretNames

	var serverRef DataMoverServerRef
	if err = Arg(args, RestoreVolumeDataDMArgDatamoverServer, &serverRef); err != nil {
		return nil, err
	}

	rvd.DataMoverServerRef = serverRef

	return rvd.RunPod(ctx)
}

func (rvd *RestoreVolumeDataDM) RunPod(ctx context.Context) (map[string]interface{}, error) {
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
		Operation:       client.FileSystemRestoreOperation{Path: rvd.DataPath, BackupID: rvd.BackupId, PVC: rvd.Volume},
		Namespace:       rvd.Namespace,
		Image:           rvd.Image,
		ServerNamespace: rvd.DataMoverServerRef.Namespace,
		ServerName:      rvd.DataMoverServerRef.Name,
		ConfigMap:       rvd.ConfigMap,
		Secrets:         rvd.Secrets,
		CredentialsConfig: client.ClientCredentialsConfig{
			CredentialsMode: client.ClientCredentialsSecret,
			SecretName:      &rvd.ClientSecret,
		},
	})

	if err != nil {
		return nil, errkit.Wrap(err, "Unable to create pod")
	}

	pc, err := rvd.runPod(ctx, cli, pod)
	if err != nil {
		return nil, errkit.Wrap(err, "Pod run error")
	}

	podOutput, err := rvd.getPodLogs(ctx, pc)
	if err != nil {
		return nil, errkit.Wrap(err, "Cannot get pod logs")
	}

	// FIXME: parse output from restore
	// FIXME: update progress percent
	output := map[string]any{
		"output": podOutput,
	}
	return output, nil
}

func (rvd *RestoreVolumeDataDM) runPod(ctx context.Context, cli kubernetes.Interface, pod *corev1.Pod) (kube.PodController, error) {
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

func (rvd *RestoreVolumeDataDM) getPodLogs(ctx context.Context, pc kube.PodController) (string, error) {
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

func (rvd *RestoreVolumeDataDM) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    rvd.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}
