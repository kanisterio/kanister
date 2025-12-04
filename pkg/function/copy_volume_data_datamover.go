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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/utils"
)

// FIXME: arg to mount PVC as read-only
const (
	CopyVolumeDataDMFuncName      = "CopyVolumeDataDM"
	CopyVolumeDataDMArgVolume     = "volume"   // TODO: PVC???
	CopyVolumeDataDMArgDataPath   = "dataPath" // TODO: dataPathPrefix???
	CopyVolumeDataDMArgTag        = "tag"      // Backup tag
	CopyVolumeDataDMArgEnv        = "env"
	CopyVolumeDataDMArgPodOptions = "podOptions"
)

type CopyVolumeDataDM struct {
	Volume          string // PVC??
	DataPath        string
	Tag             string
	dmArgs          datamoverArgs
	progressPercent string
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
		DMArgNamespace,
		DMArgImage,
		CopyVolumeDataDMArgVolume,
		DMArgDatamoverSession,
		CopyVolumeDataDMArgDataPath,
		DMArgClientSecret,
	}
}

func (cvd *CopyVolumeDataDM) Arguments() []string {
	return append(cvd.RequiredArgs(), []string{
		CopyVolumeDataDMArgTag,
		DMArgConfig,
		DMArgSecrets,
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
	if err = Arg(args, CopyVolumeDataDMArgVolume, &cvd.Volume); err != nil {
		return nil, err
	}
	if err = Arg(args, CopyVolumeDataDMArgDataPath, &cvd.DataPath); err != nil {
		return nil, err
	}

	if err = OptArg(args, CopyVolumeDataDMArgTag, &cvd.Tag, ""); err != nil {
		return nil, err
	}

	dmArgs, err := getDatamoverArgs(tp, args)
	if err != nil {
		return nil, err
	}
	cvd.dmArgs = *dmArgs

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

	// Validate PVC exists
	pvc, err := cli.CoreV1().PersistentVolumeClaims(cvd.dmArgs.namespace).Get(ctx, cvd.Volume, metav1.GetOptions{})
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to retrieve PVC.", "namespace", cvd.dmArgs.namespace, "name", cvd.Volume)
	}

	// FIXME: set owner reference for created pod (actionset)
	operation := client.FileSystemBackupOperation{
		Path:          cvd.DataPath,
		Tag:           cvd.Tag,
		PVC:           cvd.Volume,
		ReadOnlyMount: kube.PVCContainsReadOnlyAccessMode(pvc),
	}
	clientArgs := makeCreateClientArgs(cvd.dmArgs, operation)

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

func (cvd *CopyVolumeDataDM) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    cvd.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}
