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

	v1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/kube/snapshot"
	"github.com/kanisterio/kanister/pkg/param"
)

func init() {
	_ = kanister.Register(&createCSISnapshotStaticFunc{})
}

var (
	_ kanister.Func = (*createCSISnapshotStaticFunc)(nil)
)

const (
	// CreateCSISnapshotStaticFuncName gives the name of the function
	CreateCSISnapshotStaticFuncName = "CreateCSISnapshotStatic"

	// CreateCSISnapshotStaticNameArg provides name of the new VolumeSnapshot
	CreateCSISnapshotStaticNameArg = "name"

	// CreateCSISnapshotStaticNamespaceArg specifies the namespace of the new VolumeSnapshot
	CreateCSISnapshotStaticNamespaceArg = "namespace"

	// CreateCSISnapshotStaticDriverArg specifies the CSI driver used in the new VolumeSnapshotContent
	CreateCSISnapshotStaticDriverArg = "driver"

	// CreateCSISnapshotStaticSnapshotHandleArg specifies the UID of the backend storage snapshot used in the new VolumeSnapshotContent
	CreateCSISnapshotStaticSnapshotHandleArg = "snapshotHandle"

	// CreateCSISnapshotStaticSnapshotClassArg specifies the name of the VolumeSnapshotClass
	CreateCSISnapshotStaticSnapshotClassArg = "snapshotClass"

	// CreateCSISnapshotStaticOutputRestoreSize gives the storage size required for PV/PVC restoration
	CreateCSISnapshotStaticOutputRestoreSize = "restoreSize"

	// CreateCSISnapshotStaticOutputSnapshotContentName provides the name of dynamically provisioned VolumeSnapshotContent
	CreateCSISnapshotStaticOutputSnapshotContentName = "snapshotContent"
)

type createCSISnapshotStaticFunc struct{}

func (*createCSISnapshotStaticFunc) Name() string {
	return CreateCSISnapshotStaticFuncName
}

func (*createCSISnapshotStaticFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var (
		name, namespace                       string
		driver, snapshotHandle, snapshotClass string
	)

	if err := Arg(args, CreateCSISnapshotStaticNameArg, &name); err != nil {
		return nil, err
	}

	if err := Arg(args, CreateCSISnapshotStaticNamespaceArg, &namespace); err != nil {
		return nil, err
	}

	if err := Arg(args, CreateCSISnapshotStaticDriverArg, &driver); err != nil {
		return nil, err
	}

	if err := Arg(args, CreateCSISnapshotStaticSnapshotHandleArg, &snapshotHandle); err != nil {
		return nil, err
	}

	if err := Arg(args, CreateCSISnapshotStaticSnapshotClassArg, &snapshotClass); err != nil {
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

	// waitForReady is set to true by default because snapshot information is needed as output artifacts
	waitForReady := true
	vs, err := createCSISnapshotStatic(ctx, snapshotter, name, namespace, driver, snapshotHandle, snapshotClass, waitForReady)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		CreateCSISnapshotStaticNameArg:                name,
		CreateCSISnapshotStaticNamespaceArg:           namespace,
		CreateCSISnapshotStaticOutputRestoreSize:         vs.Status.RestoreSize.String(),
		CreateCSISnapshotStaticOutputSnapshotContentName: vs.Status.BoundVolumeSnapshotContentName,
	}, nil
}

func (*createCSISnapshotStaticFunc) RequiredArgs() []string {
	return []string{
		CreateCSISnapshotStaticNameArg,
		CreateCSISnapshotStaticNamespaceArg,
		CreateCSISnapshotStaticDriverArg,
		CreateCSISnapshotStaticSnapshotHandleArg,
		CreateCSISnapshotStaticSnapshotClassArg,
	}
}

func (*createCSISnapshotStaticFunc) Arguments() []string {
	return []string{
		CreateCSISnapshotStaticNameArg,
		CreateCSISnapshotStaticNamespaceArg,
		CreateCSISnapshotStaticDriverArg,
		CreateCSISnapshotStaticSnapshotHandleArg,
		CreateCSISnapshotStaticSnapshotClassArg,
	}
}

func createCSISnapshotStatic(
	ctx context.Context,
	snapshotter snapshot.Snapshotter,
	name, namespace, driver, snapshotHandle, snapshotClass string,
	wait bool,
) (*v1.VolumeSnapshot, error) {
	source := &snapshot.Source{
		Handle:                  snapshotHandle,
		Driver:                  driver,
		VolumeSnapshotClassName: snapshotClass,
	}
	if err := snapshotter.CreateFromSource(ctx, source, name, namespace, wait); err != nil {
		return nil, err
	}

	return snapshotter.Get(ctx, name, namespace)
}
