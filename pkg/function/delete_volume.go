// Copyright 2024 The Kanister Authors.
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
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/utils"
)

func init() {
	_ = kanister.Register(&deleteVolumeFunc{})
}

var _ kanister.Func = (*deleteVolumeFunc)(nil)

const (
	DeleteVolumeFuncName         = "DeleteVolume"
	DeleteVolumePVCNameArg       = "pvcName"
	DeleteVolumePVCNamespaceArg  = "pvcNamespace"
)

type deleteVolumeFunc struct {
	progressPercent string
}

func (*deleteVolumeFunc) Name() string {
	return DeleteVolumeFuncName
}

func (d *deleteVolumeFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	d.progressPercent = progress.StartedPercent
	defer func() { d.progressPercent = progress.CompletedPercent }()

	var pvcName, pvcNamespace string
	if err := Arg(args, DeleteVolumePVCNameArg, &pvcName); err != nil {
		return nil, err
	}
	if err := Arg(args, DeleteVolumePVCNamespaceArg, &pvcNamespace); err != nil {
		return nil, err
	}

	kubeCli, err := kube.NewClient()
	if err != nil {
		return nil, err
	}

	err = kubeCli.CoreV1().PersistentVolumeClaims(pvcNamespace).Delete(ctx, pvcName, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}
	return nil, nil
}

func (*deleteVolumeFunc) RequiredArgs() []string {
	return []string{
		DeleteVolumePVCNameArg,
		DeleteVolumePVCNamespaceArg,
	}
}

func (*deleteVolumeFunc) Arguments() []string {
	return []string{
		DeleteVolumePVCNameArg,
		DeleteVolumePVCNamespaceArg,
	}
}

func (d *deleteVolumeFunc) Validate(args map[string]any) error {
	if err := utils.CheckSupportedArgs(d.Arguments(), args); err != nil {
		return err
	}
	return utils.CheckRequiredArgs(d.RequiredArgs(), args)
}

func (d *deleteVolumeFunc) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    d.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}
