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
	"time"

	"github.com/kanisterio/datamover/pkg/client"
	"github.com/kanisterio/errkit"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/output"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/utils"
)

const (
	StreamRestoreDMFuncName            = "StreamRestoreDM"
	StreamRestoreDMArgDatamoverSession = "datamoverSession"
	StreamRestoreDMArgIngestor         = "streamIngestor"
	StreamRestoreDMArgBackupObjectName = "backupObjectName"
	StreamRestoreDMArgBackupID         = "backupId" // Backup tag // FIXME: make optional?
	StreamRestoreDMArgInitImage        = "initImage"
	StreamRestoreDMArgClientSecret     = "clientSecret" // TODO: clientSecretVolume???
	StreamRestoreDMArgConfig           = "config"
	StreamRestoreDMArgSecrets          = "secrets"
)

type StreamRestoreDM struct {
	StreamIngestor   corev1.Container
	BackupObjectName string
	BackupID         string
	InitImage        string
	dmArgs           datamoverArgs
	progressPercent  string
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
		DMArgNamespace,
		DMArgImage,
		DMArgDatamoverSession,
		StreamRestoreDMArgIngestor,
		StreamRestoreDMArgBackupID,
		DMArgClientSecret,
		// TODO: implementation specific secrets
		// TLS fingerprint secret
	}
}

func (streamRestore *StreamRestoreDM) Arguments() []string {
	return append(streamRestore.RequiredArgs(), []string{
		StreamRestoreDMArgBackupObjectName,
		StreamRestoreDMArgInitImage,
		DMArgConfig,
		DMArgSecrets,
		DMArgEnv,
		DMArgPodOptions,
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
	var generatorContainer corev1.Container
	if err = Arg(args, StreamRestoreDMArgIngestor, &generatorContainer); err != nil {
		return nil, err
	}
	streamRestore.StreamIngestor = generatorContainer

	if err = OptArg(args, StreamRestoreDMArgBackupID, &streamRestore.BackupID, ""); err != nil {
		return nil, err
	}

	if err = OptArg(args, StreamRestoreDMArgBackupObjectName, &streamRestore.BackupObjectName, "data"); err != nil {
		return nil, err
	}

	if err = OptArg(args, StreamRestoreDMArgInitImage, &streamRestore.InitImage, ""); err != nil {
		return nil, err
	}

	dmArgs, err := getDatamoverArgs(tp, args)
	if err != nil {
		return nil, err
	}
	streamRestore.dmArgs = *dmArgs

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

	operation := client.StreamRestoreOperation{
		BackupID:         streamRestore.BackupID,
		BackupObjectName: streamRestore.BackupObjectName,
		InitImage:        streamRestore.InitImage,
		StreamIngestor:   streamRestore.StreamIngestor,
	}

	clientArgs := makeCreateClientArgs(streamRestore.dmArgs, operation)

	pod, err := client.CreateClientPod(ctx, cli, dynCli, clientArgs)

	if err != nil {
		return nil, errkit.Wrap(err, "Unable to create pod")
	}

	err = monitorDatamoverPod(ctx, cli, pod)
	if err != nil {
		return nil, errkit.Wrap(err, "Pod run error")
	}

	podOutput, err := streamPodLogs(ctx, cli, *pod, streamRestore.StreamIngestor.Name)
	if err != nil {
		return nil, errkit.Wrap(err, "Cannot get pod logs")
	}

	out, err := output.LogAndParse(ctx, podOutput)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (streamRestore *StreamRestoreDM) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    streamRestore.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}
