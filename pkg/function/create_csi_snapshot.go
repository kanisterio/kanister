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
	"fmt"
	"time"

	v1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/kube/snapshot"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/utils"
)

func init() {
	_ = kanister.Register(&createCSISnapshotFunc{})
}

var (
	_ kanister.Func = (*createCSISnapshotFunc)(nil)
)

const (
	// CreateCSIVolumeSnapshotFuncName gives the name of the function
	CreateCSISnapshotFuncName = "CreateCSISnapshot"
	// CreateCSISnapshotNameArg provides name of the new VolumeSnapshot
	CreateCSISnapshotNameArg = "name"
	// CreateCSISnapshotPVCNameArg gives the name of the captured PVC
	CreateCSISnapshotPVCNameArg = "pvc"
	// CreateCSISnapshotNamespaceArg mentions the namespace of the captured PVC
	CreateCSISnapshotNamespaceArg = "namespace"
	// CreateCSISnapshotSnapshotClassArg specifies the name of the VolumeSnapshotClass
	CreateCSISnapshotSnapshotClassArg = "snapshotClass"
	// CreateCSISnapshotLabelsArg has labels that are to be added to the new VolumeSnapshot
	CreateCSISnapshotLabelsArg = "labels"
	// CreateCSISnapshotRestoreSizeArg gives the storage size required for PV/PVC restoration
	CreateCSISnapshotRestoreSizeArg = "restoreSize"
	// CreateCSISnapshotSnapshotContentNameArg provides the name of dynamically provisioned VolumeSnapshotContent
	CreateCSISnapshotSnapshotContentNameArg = "snapshotContent"
)

type createCSISnapshotFunc struct {
	progressPercent string
}

func (*createCSISnapshotFunc) Name() string {
	return CreateCSISnapshotFuncName
}

func (c *createCSISnapshotFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	// Set progress percent
	c.progressPercent = progress.StartedPercent
	defer func() { c.progressPercent = progress.CompletedPercent }()

	var snapshotClass string
	var labels map[string]string
	var name, pvc, namespace string
	if err := Arg(args, CreateCSISnapshotPVCNameArg, &pvc); err != nil {
		return nil, err
	}
	if err := Arg(args, CreateCSISnapshotNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err := Arg(args, CreateCSISnapshotSnapshotClassArg, &snapshotClass); err != nil {
		return nil, err
	}
	if err := OptArg(args, CreateCSISnapshotNameArg, &name, defaultSnapshotName(pvc, 5)); err != nil {
		return nil, err
	}
	if err := OptArg(args, CreateCSISnapshotLabelsArg, &labels, map[string]string{}); err != nil {
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
		if errors.Is(context.DeadlineExceeded, err) {
			timeoutMsg := "SnapshotContent not provisioned within given timeout. Please check if CSI driver is installed correctly and supports VolumeSnapshot feature"
			return nil, errors.Wrap(err, timeoutMsg)
		}
		return nil, err
	}
	// waitForReady is set to true by default because snapshot information is needed as output artifacts
	waitForReady := true
	vs, err := createCSISnapshot(ctx, snapshotter, name, namespace, pvc, snapshotClass, waitForReady, labels)
	if err != nil {
		return nil, err
	}

	snapshotInfo := map[string]interface{}{
		CreateCSISnapshotNameArg:                name,
		CreateCSISnapshotPVCNameArg:             pvc,
		CreateCSISnapshotNamespaceArg:           namespace,
		CreateCSISnapshotRestoreSizeArg:         vs.Status.RestoreSize.String(),
		CreateCSISnapshotSnapshotContentNameArg: vs.Status.BoundVolumeSnapshotContentName,
	}
	return snapshotInfo, nil
}

func (*createCSISnapshotFunc) RequiredArgs() []string {
	return []string{
		CreateCSISnapshotPVCNameArg,
		CreateCSISnapshotNamespaceArg,
		CreateCSISnapshotSnapshotClassArg,
	}
}

func (*createCSISnapshotFunc) Arguments() []string {
	return []string{
		CreateCSISnapshotPVCNameArg,
		CreateCSISnapshotNamespaceArg,
		CreateCSISnapshotSnapshotClassArg,
		CreateCSISnapshotNameArg,
		CreateCSISnapshotLabelsArg,
	}
}

func (c *createCSISnapshotFunc) Validate(args map[string]any) error {
	if err := utils.CheckSupportedArgs(c.Arguments(), args); err != nil {
		return err
	}

	return utils.CheckRequiredArgs(c.RequiredArgs(), args)
}

func createCSISnapshot(ctx context.Context, snapshotter snapshot.Snapshotter, name, namespace, pvc, snapshotClass string, wait bool, labels map[string]string) (*v1.VolumeSnapshot, error) {
	snapshotMeta := snapshot.ObjectMeta{
		Name:        name,
		Namespace:   namespace,
		Labels:      labels,
		Annotations: nil,
	}
	if err := snapshotter.Create(ctx, pvc, &snapshotClass, wait, snapshotMeta); err != nil {
		return nil, err
	}
	vs, err := snapshotter.Get(ctx, name, namespace)
	if err != nil {
		return nil, err
	}
	return vs, nil
}

// defaultSnapshotName generates snapshot name using <pvcName>-snapshot-<randomValue>
func defaultSnapshotName(pvcName string, len int) string {
	return fmt.Sprintf("%s-snapshot-%s", pvcName, rand.String(len))
}

func (c *createCSISnapshotFunc) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    c.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}
