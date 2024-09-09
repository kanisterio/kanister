// Copyright 2019 The Kanister Authors.
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
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/restic"
	"github.com/kanisterio/kanister/pkg/utils"
)

const (
	restoreDataAllJobPrefix = "restore-data-all-"
	// RestoreDataAllFuncName gives the function name
	RestoreDataAllFuncName = "RestoreDataAll"
	// RestoreDataAllNamespaceArg provides the namespace
	RestoreDataAllNamespaceArg = "namespace"
	// RestoreDataAllImageArg provides the image of the container with required tools
	RestoreDataAllImageArg = "image"
	// RestoreDataAllBackupArtifactPrefixArg provides the path of the backed up artifact
	RestoreDataAllBackupArtifactPrefixArg = "backupArtifactPrefix"
	// RestoreDataAllRestorePathArg provides the path to restore backed up data
	RestoreDataAllRestorePathArg = "restorePath"
	// RestoreDataAllPodsArg provides the pod connected to the data volume
	RestoreDataAllPodsArg = "pods"
	// RestoreDataAllEncryptionKeyArg provides the encryption key used during backup
	RestoreDataAllEncryptionKeyArg = "encryptionKey"
	// RestoreDataAllBackupInfo provides backup info required for restore
	RestoreDataAllBackupInfo = "backupInfo"
	// RestoreDataPodOverrideArg contains pod specs which overrides default pod specs
	RestoreDataAllPodOverrideArg = "podOverride"
)

func init() {
	_ = kanister.Register(&restoreDataAllFunc{})
}

var _ kanister.Func = (*restoreDataAllFunc)(nil)

type restoreDataAllFunc struct {
	progressPercent string
}

func (*restoreDataAllFunc) Name() string {
	return RestoreDataAllFuncName
}

func validateAndGetRestoreAllOptArgs(args map[string]interface{}, tp param.TemplateParams) (string, string, []string, bool, crv1alpha1.JSONMap, error) {
	var restorePath, encryptionKey, pods string
	var ps []string
	var podOverride crv1alpha1.JSONMap
	var err error
	var insecureTLS bool

	if err = OptArg(args, RestoreDataAllRestorePathArg, &restorePath, "/"); err != nil {
		return restorePath, encryptionKey, ps, insecureTLS, podOverride, err
	}
	if err = OptArg(args, RestoreDataAllEncryptionKeyArg, &encryptionKey, restic.GeneratePassword()); err != nil {
		return restorePath, encryptionKey, ps, insecureTLS, podOverride, err
	}
	if err = OptArg(args, RestoreDataAllPodsArg, &pods, ""); err != nil {
		return restorePath, encryptionKey, ps, insecureTLS, podOverride, err
	}
	if err = OptArg(args, InsecureTLS, &insecureTLS, false); err != nil {
		return restorePath, encryptionKey, ps, insecureTLS, podOverride, err
	}
	podOverride, err = GetPodSpecOverride(tp, args, RestoreDataAllPodOverrideArg)
	if err != nil {
		return restorePath, encryptionKey, ps, insecureTLS, podOverride, err
	}

	if pods != "" {
		ps = strings.Fields(pods)
	} else {
		switch {
		case tp.Deployment != nil:
			ps = tp.Deployment.Pods
		case tp.StatefulSet != nil:
			ps = tp.StatefulSet.Pods
		default:
			return restorePath, encryptionKey, ps, insecureTLS, podOverride, errors.New("Unsupported workload type")
		}
	}

	return restorePath, encryptionKey, ps, insecureTLS, podOverride, err
}

func (r *restoreDataAllFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	// Set progress percent
	r.progressPercent = progress.StartedPercent
	defer func() { r.progressPercent = progress.CompletedPercent }()

	var namespace, image, backupArtifactPrefix, backupInfo string
	var err error
	var bpAnnotations, bpLabels map[string]string

	if err = Arg(args, RestoreDataAllNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, RestoreDataAllImageArg, &image); err != nil {
		return nil, err
	}
	if err = Arg(args, RestoreDataAllBackupArtifactPrefixArg, &backupArtifactPrefix); err != nil {
		return nil, err
	}
	if err = Arg(args, RestoreDataAllBackupInfo, &backupInfo); err != nil {
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

	// Validate and get optional arguments
	restorePath, encryptionKey, pods, insecureTLS, podOverride, err := validateAndGetRestoreAllOptArgs(args, tp)
	if err != nil {
		return nil, err
	}

	if err = ValidateProfile(tp.Profile); err != nil {
		return nil, err
	}
	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	input := make(map[string]BackupInfo)
	err = json.Unmarshal([]byte(backupInfo), &input)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not decode JSON data")
	}
	var chanLen = len(pods)
	errChan := make(chan error, chanLen)
	outputChan := make(chan map[string]interface{}, chanLen)
	output := make(map[string]interface{})
	for _, pod := range pods {
		go func(pod string) {
			vols, err := FetchPodVolumes(pod, tp)
			var out map[string]interface{}
			if err != nil {
				errChan <- errors.Wrapf(err, "Failed to get volumes of pod %s", pod)
				outputChan <- out
				return
			}
			out, err = restoreData(
				ctx,
				cli,
				tp,
				namespace,
				encryptionKey,
				fmt.Sprintf("%s/%s", backupArtifactPrefix, pod),
				restorePath,
				"",
				input[pod].BackupID,
				restoreDataAllJobPrefix,
				image,
				insecureTLS,
				vols,
				podOverride,
				annotations,
				labels,
			)
			errChan <- errors.Wrapf(err, "Failed to restore data for pod %s", pod)
			outputChan <- out
		}(pod)
	}
	errs := make([]string, 0, chanLen)
	for i := 0; i < chanLen; i++ {
		err := <-errChan
		out := <-outputChan
		if err != nil {
			errs = append(errs, err.Error())
		} else {
			for k, v := range out {
				output[k] = v
			}
		}
	}
	if len(errs) != 0 {
		return nil, errors.New(strings.Join(errs, "\n"))
	}
	return output, nil
}

func (*restoreDataAllFunc) RequiredArgs() []string {
	return []string{
		RestoreDataAllNamespaceArg,
		RestoreDataAllImageArg,
		RestoreDataAllBackupArtifactPrefixArg,
		RestoreDataAllBackupInfo,
	}
}

func (*restoreDataAllFunc) Arguments() []string {
	return []string{
		RestoreDataAllNamespaceArg,
		RestoreDataAllImageArg,
		RestoreDataAllBackupArtifactPrefixArg,
		RestoreDataAllBackupInfo,
		RestoreDataAllRestorePathArg,
		RestoreDataAllEncryptionKeyArg,
		RestoreDataAllPodsArg,
		RestoreDataAllPodOverrideArg,
		InsecureTLS,
		PodAnnotationsArg,
		PodLabelsArg,
	}
}

func (r *restoreDataAllFunc) Validate(args map[string]any) error {
	if err := ValidatePodLabelsAndAnnotations(r.Name(), args); err != nil {
		return err
	}

	if err := utils.CheckSupportedArgs(r.Arguments(), args); err != nil {
		return err
	}

	return utils.CheckRequiredArgs(r.RequiredArgs(), args)
}

func (r *restoreDataAllFunc) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    r.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}
