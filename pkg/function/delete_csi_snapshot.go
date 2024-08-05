// Copyright 2022 The Kanister Authors.
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

	v1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/kube/snapshot"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/poll"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/utils"
)

func init() {
	_ = kanister.Register(&deleteCSISnapshotFunc{})
}

var (
	_ kanister.Func = (*deleteCSISnapshotFunc)(nil)
)

const (
	// DeleteCSIVolumeSnapshotFuncName gives the name of the function
	DeleteCSISnapshotFuncName = "DeleteCSISnapshot"
	// DeleteCSISnapshotNameArg provides name of the VolumeSnapshot that needs to be deleted
	DeleteCSISnapshotNameArg = "name"
	// DeleteCSISnapshotNamespaceArg mentions the namespace where the VolumeSnapshot resides
	DeleteCSISnapshotNamespaceArg = "namespace"
)

type deleteCSISnapshotFunc struct {
	progressPercent string
}

func (*deleteCSISnapshotFunc) Name() string {
	return DeleteCSISnapshotFuncName
}

func (d *deleteCSISnapshotFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	// Set progress percent
	d.progressPercent = progress.StartedPercent
	defer func() { d.progressPercent = progress.CompletedPercent }()

	var name, namespace string
	if err := Arg(args, DeleteCSISnapshotNameArg, &name); err != nil {
		return nil, err
	}
	if err := Arg(args, DeleteCSISnapshotNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	kubeCli, err := kube.NewClient()
	if err != nil {
		return nil, err
	}
	dynCli, err := kube.NewDynamicClient()
	if err != nil {
		return nil, err
	}
	snapshotter, err := snapshot.NewSnapshotter(kubeCli, dynCli)
	if err != nil {
		return nil, err
	}
	if _, err := deleteCSISnapshot(ctx, snapshotter, name, namespace); err != nil {
		return nil, err
	}
	if err := waitForCSISnapshotDeletion(ctx, snapshotter, name, namespace); err != nil {
		return nil, err
	}
	return nil, nil
}

func (*deleteCSISnapshotFunc) RequiredArgs() []string {
	return []string{
		DeleteCSISnapshotNameArg,
		DeleteCSISnapshotNamespaceArg,
	}
}

func (*deleteCSISnapshotFunc) Arguments() []string {
	return []string{
		DeleteCSISnapshotNameArg,
		DeleteCSISnapshotNamespaceArg,
	}
}

func (d *deleteCSISnapshotFunc) Validate(args map[string]any) error {
	if err := utils.CheckSupportedArgs(d.Arguments(), args); err != nil {
		return err
	}

	return utils.CheckRequiredArgs(d.RequiredArgs(), args)
}

func (c *deleteCSISnapshotFunc) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    c.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}

func deleteCSISnapshot(ctx context.Context, snapshotter snapshot.Snapshotter, name, namespace string) (*v1.VolumeSnapshot, error) {
	return snapshotter.Delete(ctx, name, namespace)
}

func waitForCSISnapshotDeletion(ctx context.Context, snapshotter snapshot.Snapshotter, name, namespace string) error {
	return poll.Wait(ctx, func(context.Context) (done bool, err error) {
		_, err = snapshotter.Get(ctx, name, namespace)
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
}
