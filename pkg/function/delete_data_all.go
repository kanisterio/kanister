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
	// DeleteDataAllFuncName gives the name of the function
	DeleteDataAllFuncName = "DeleteDataAll"
	// DeleteDataAllNamespaceArg provides the namespace
	DeleteDataAllNamespaceArg = "namespace"
	// DeleteDataAllBackupArtifactPrefixArg provides the path to restore backed up data
	DeleteDataAllBackupArtifactPrefixArg = "backupArtifactPrefix"
	// DeleteDataAllEncryptionKeyArg provides the encryption key to be used for deletes
	DeleteDataAllEncryptionKeyArg = "encryptionKey"
	// DeleteDataAllReclaimSpace provides a way to specify if space should be reclaimed
	DeleteDataAllReclaimSpace = "reclaimSpace"
	// DeleteDataAllBackupInfo provides backup info required for delete
	DeleteDataAllBackupInfo = "backupInfo"
	// DeleteDataAllPodOverrideArg contains pod specs to override default pod specs
	DeleteDataAllPodOverrideArg = "podOverride"
	deleteDataAllJobPrefix      = "delete-data-all-"
)

func init() {
	_ = kanister.Register(&deleteDataAllFunc{})
}

var _ kanister.Func = (*deleteDataAllFunc)(nil)

type deleteDataAllFunc struct {
	progressPercent string
}

func (*deleteDataAllFunc) Name() string {
	return DeleteDataAllFuncName
}

func (d *deleteDataAllFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	// Set progress percent
	d.progressPercent = progress.StartedPercent
	defer func() { d.progressPercent = progress.CompletedPercent }()

	var namespace, deleteArtifactPrefix, backupInfo, encryptionKey string
	var reclaimSpace bool
	var err error
	var insecureTLS bool
	var bpAnnotations, bpLabels map[string]string
	if err = Arg(args, DeleteDataAllNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, DeleteDataAllBackupArtifactPrefixArg, &deleteArtifactPrefix); err != nil {
		return nil, err
	}
	if err = Arg(args, DeleteDataAllBackupInfo, &backupInfo); err != nil {
		return nil, err
	}
	if err = OptArg(args, DeleteDataAllEncryptionKeyArg, &encryptionKey, restic.GeneratePassword()); err != nil {
		return nil, err
	}
	if err = OptArg(args, DeleteDataAllReclaimSpace, &reclaimSpace, false); err != nil {
		return nil, err
	}
	if err = OptArg(args, InsecureTLS, &insecureTLS, false); err != nil {
		return nil, err
	}
	if err = OptArg(args, PodAnnotationsArg, &bpAnnotations, nil); err != nil {
		return nil, err
	}
	if err = OptArg(args, PodLabelsArg, &bpLabels, nil); err != nil {
		return nil, err
	}
	podOverride, err := GetPodSpecOverride(tp, args, DeleteDataAllPodOverrideArg)
	if err != nil {
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
	var targetPaths []string
	var deleteIdentifiers []string
	for pod, info := range input {
		targetPaths = append(targetPaths, fmt.Sprintf("%s/%s", deleteArtifactPrefix, pod))
		deleteIdentifiers = append(deleteIdentifiers, info.BackupID)
	}

	return deleteData(
		ctx,
		cli,
		tp,
		reclaimSpace,
		namespace,
		encryptionKey,
		insecureTLS,
		targetPaths,
		nil,
		deleteIdentifiers,
		deleteDataAllJobPrefix,
		podOverride,
		annotations,
		labels,
	)
}

func (*deleteDataAllFunc) RequiredArgs() []string {
	return []string{
		DeleteDataAllNamespaceArg,
		DeleteDataAllBackupArtifactPrefixArg,
		DeleteDataAllBackupInfo,
	}
}

func (*deleteDataAllFunc) Arguments() []string {
	return []string{
		DeleteDataAllNamespaceArg,
		DeleteDataAllBackupArtifactPrefixArg,
		DeleteDataAllBackupInfo,
		DeleteDataAllEncryptionKeyArg,
		DeleteDataAllReclaimSpace,
		InsecureTLS,
		PodAnnotationsArg,
		PodLabelsArg,
	}
}

func (d *deleteDataAllFunc) Validate(args map[string]any) error {
	if err := ValidatePodLabelsAndAnnotations(d.Name(), args); err != nil {
		return err
	}

	if err := utils.CheckSupportedArgs(d.Arguments(), args); err != nil {
		return err
	}

	return utils.CheckRequiredArgs(d.RequiredArgs(), args)
}

func (d *deleteDataAllFunc) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    d.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}
