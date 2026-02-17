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
	"github.com/kanisterio/kanister/pkg/output"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/utils"
)

const (
	RestoreVolumeDataDMFuncName    = "RestoreVolumeDataDM"
	RestoreVolumeDataDMArgVolume   = "volume"   // TODO: PVC???
	RestoreVolumeDataDMArgDataPath = "dataPath" // TODO: dataPathPrefix???
	RestoreVolumeDataDMArgBackupID = "backupID" // Backup id
)

type RestoreVolumeDataDM struct {
	Volume          string // PVC??
	DataPath        string
	BackupID        string
	dmArgs          datamoverArgs
	progressPercent string
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
		DMArgNamespace,
		DMArgImage,
		RestoreVolumeDataDMArgVolume,
		DMArgDatamoverSession,
		RestoreVolumeDataDMArgDataPath,
		RestoreVolumeDataDMArgBackupID,
		DMArgClientSecret,
		// TODO: implementation specific secrets
		// TLS fingerprint secret
	}
}

func (rvd *RestoreVolumeDataDM) Arguments() []string {
	return append(rvd.RequiredArgs(), []string{
		DMArgConfig,
		DMArgSecrets,
		DMArgEnv,
		DMArgPodOptions,
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
	if err = Arg(args, RestoreVolumeDataDMArgVolume, &rvd.Volume); err != nil {
		return nil, err
	}
	if err = Arg(args, RestoreVolumeDataDMArgDataPath, &rvd.DataPath); err != nil {
		return nil, err
	}

	if err = OptArg(args, RestoreVolumeDataDMArgBackupID, &rvd.BackupID, ""); err != nil {
		return nil, err
	}

	dmArgs, err := getDatamoverArgs(tp, args)
	if err != nil {
		return nil, err
	}
	rvd.dmArgs = *dmArgs

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

	operation := client.FileSystemRestoreOperation{Path: rvd.DataPath, BackupID: rvd.BackupID, PVC: rvd.Volume}
	clientArgs := makeCreateClientArgs(rvd.dmArgs, operation)

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

	out, err := output.LogAndParse(ctx, podOutputReader)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (rvd *RestoreVolumeDataDM) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    rvd.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}
