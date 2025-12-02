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
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/utils"
)

const (
	StreamBackupDMFuncName            = "StreamBackupDM"
	StreamBackupDMArgGenerator        = "streamGenerator"
	StreamBackupDMArgBackupObjectName = "backupObjectName"
	StreamBackupDMArgTag              = "tag" // Backup tag
	StreamBackupDMArgInitImage        = "initImage"
)

type StreamBackupDM struct {
	StreamGenerator  corev1.Container
	BackupObjectName string
	Tag              string
	InitImage        string
	dmArgs           datamoverArgs
	progressPercent  string
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
		DMArgNamespace,
		DMArgImage,
		DMArgDatamoverSession,
		StreamBackupDMArgGenerator,
		DMArgClientSecret,
		// TODO: implementation specific secrets
		// TLS fingerprint secret
	}
}

func (streamBackup *StreamBackupDM) Arguments() []string {
	return append(streamBackup.RequiredArgs(), []string{
		StreamBackupDMArgTag,
		StreamBackupDMArgBackupObjectName,
		StreamBackupDMArgInitImage,
		DMArgConfig,
		DMArgSecrets,
		DMArgEnv,
		DMArgPodOptions,
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
	var generatorContainer corev1.Container
	if err = Arg(args, StreamBackupDMArgGenerator, &generatorContainer); err != nil {
		return nil, err
	}
	streamBackup.StreamGenerator = generatorContainer

	if err = OptArg(args, StreamBackupDMArgTag, &streamBackup.Tag, ""); err != nil {
		return nil, err
	}

	if err = OptArg(args, StreamBackupDMArgBackupObjectName, &streamBackup.BackupObjectName, "data"); err != nil {
		return nil, err
	}

	if err = OptArg(args, StreamBackupDMArgInitImage, &streamBackup.InitImage, ""); err != nil {
		return nil, err
	}

	dmArgs, err := getDatamoverArgs(tp, args)
	if err != nil {
		return nil, err
	}
	streamBackup.dmArgs = *dmArgs

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
	operation := client.StreamBackupOperation{
		Tag:              streamBackup.Tag,
		StreamGenerator:  streamBackup.StreamGenerator,
		BackupObjectName: streamBackup.BackupObjectName,
		InitImage:        streamBackup.InitImage}

	clientArgs := makeCreateClientArgs(streamBackup.dmArgs, operation)

	pod, err := client.CreateClientPod(ctx, cli, dynCli, clientArgs)

	if err != nil {
		return nil, errkit.Wrap(err, "Unable to create pod")
	}

	err = monitorDatamoverPod(ctx, cli, pod)
	if err != nil {
		return nil, errkit.Wrap(err, "Pod run error")
	}

	podOutputReader, err := streamPodLogs(ctx, cli, *pod, client.MainContainerName)
	if err != nil {
		return nil, errkit.Wrap(err, "Cannot get pod logs")
	}

	// FIXME: this parsing is kopia specific
	// Implementation should output information in generic format instead
	kopiaOutput, err := parseKopiaOutput(podOutputReader)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to read kopia output from the pod")
	}

	return kopiaOutput, nil
}

func (streamBackup *StreamBackupDM) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    streamBackup.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}
