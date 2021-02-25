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

package snapshot

import (
	"context"

	v1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	"github.com/pkg/errors"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/kube/snapshot/apis/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube/snapshot/apis/v1beta1"
)

type Snapshotter interface {
	// GetVolumeSnapshotClass returns VolumeSnapshotClass name which is annotated with given key
	//
	// 'annotationKey' is the annotation key which has to be present on VolumeSnapshotClass.
	// 'annotationValue' is the value for annotationKey in VolumeSnapshotClass spec.
	// 'storageClassName' is the name of the storageClass that shares the same driver as the VolumeSnapshotClass.
	// This returns error if no VolumeSnapshotClass found.
	GetVolumeSnapshotClass(annotationKey, annotationValue, storageClassName string) (string, error)

	// CloneVolumeSnapshotClass creates a copy of the source volume snapshot
	// class with the specified deletion policy and name. If the target
	// already exists, it returns no error.
	//
	// 'sourceClassName' is the name of the source VolumeSnapshotClass.
	// 'targetClassName' is the name of the target VolumeSnapshotClass
	// 'newDeletionPolicy' is the deletion policy to set on the target.
	// 'excludeAnnotations' are the annotations that should not be set on the
	// target
	CloneVolumeSnapshotClass(sourceClassName, targetClassName, newDeletionPolicy string, excludeAnnotations []string) error
	// Create creates a VolumeSnapshot and returns it or any error happened meanwhile.
	//
	// 'name' is the name of the VolumeSnapshot.
	// 'namespace' is namespace of the PVC. VolumeSnapshot will be crated in the same namespace.
	// 'pvcName' is the name of the PVC of which we will take snapshot. It must be in the same namespace 'ns'.
	// 'waitForReady' will block the caller until the snapshot status is 'ReadyToUse'.
	// or 'ctx.Done()' is signalled. Otherwise it will return immediately after the snapshot is cut.
	// 'labels' can also be addded to the volume snapshot.
	Create(ctx context.Context, name, namespace, pvcName string, snapshotClass *string, waitForReady bool, labels map[string]string) error
	// Get will return the VolumeSnapshot in the namespace 'namespace' with given 'name'.
	//
	// 'name' is the name of the VolumeSnapshot that will be returned.
	// 'namespace' is the namespace of the VolumeSnapshot that will be returned.
	Get(ctx context.Context, name, namespace string) (*v1.VolumeSnapshot, error)
	// Delete will delete the VolumeSnapshot.
	// Returns the `VolumeSnapshot` deleted and any error as a result.
	//
	// 'name' is the name of the VolumeSnapshot that will be deleted.
	// 'namespace' is the namespace of the VolumeSnapshot that will be deleted.
	Delete(ctx context.Context, name, namespace string) (*v1.VolumeSnapshot, error)
	// DeleteContent will delete the VolumeSnapshot and returns any error as a
	// result.
	//
	// 'name' is the name of the VolumeSnapshotContent that will be deleted.
	DeleteContent(ctx context.Context, name string) error
	// Clone will clone the VolumeSnapshot to namespace 'cloneNamespace'.
	// Underlying VolumeSnapshotContent will be cloned with a different name.
	//
	// 'name' is the name of the VolumeSnapshot that will be cloned.
	// 'namespace' is the namespace of the VolumeSnapshot that will be cloned.
	// 'cloneName' is name of the clone.
	// 'cloneNamespace' is the namespace where the clone will be created.
	// 'waitForReady' will make the function blocks until the clone's status is ready to use.
	Clone(ctx context.Context, name, namespace, cloneName, cloneNamespace string, waitForReady bool) error
	// GetSource will return the CSI source that backs the volume snapshot.
	//
	// 'snapshotName' is the name of the Volumesnapshot.
	// 'namespace' is the namespace of the Volumesnapshot.
	GetSource(ctx context.Context, snapshotName, namespace string) (*Source, error)
	// CreateFromSource will create a 'Volumesnapshot' and 'VolumesnaphotContent' pair for the underlying snapshot source.
	//
	// 'source' contains information about CSI snapshot.
	// 'snapshotName' is the name of the snapshot that will be created.
	// 'namespace' is the namespace of the snapshot.
	// 'waitForReady' blocks the caller until snapshot is ready to use or context is cancelled.
	CreateFromSource(ctx context.Context, source *Source, snapshotName, namespace string, waitForReady bool) error
	// CreateContentFromSource will create a 'VolumesnaphotContent' for the underlying snapshot source.
	//
	// 'source' contains information about CSI snapshot.
	// 'contentName' is the name of the VSC that will be created
	// 'snapshotName' is the name of the snapshot that will be reference the VSC
	// 'namespace' is the namespace of the snapshot.
	// 'deletionPolicy' is the deletion policy to set on the created VSC
	CreateContentFromSource(ctx context.Context, source *Source, contentName, snapshotName, namespace, deletionPolicy string) error
	// WaitOnReadyToUse will block until the Volumesnapshot in namespace 'namespace' with name 'snapshotName'
	// has status 'ReadyToUse' or 'ctx.Done()' is signalled.
	WaitOnReadyToUse(ctx context.Context, snapshotName, namespace string) error
	// List will list the volumesnapshots in a namespace that match search. If labels aren't provided,
	// it will list all the snapshots in the namespace
	List(ctx context.Context, namespace string, labels map[string]string) (*v1.VolumeSnapshotList, error)
}

// Source represents the CSI source of the Volumesnapshot.
type Source struct {
	Handle                  string
	Driver                  string
	RestoreSize             *int64
	VolumeSnapshotClassName string
}

// NewSnapshotter creates and return new Snapshotter object
func NewSnapshotter(kubeCli kubernetes.Interface, dynCli dynamic.Interface) (Snapshotter, error) {
	ctx := context.Background()
	// Check if v1alpha1 snapshot API exists
	exists, err := kube.IsGroupVersionAvailable(ctx, kubeCli.Discovery(), v1alpha1.GroupName, v1alpha1.Version)
	if err != nil {
		return nil, errors.Errorf("Failed to call discovery APIs: %v", err)
	}
	if exists {
		return NewSnapshotAlpha(kubeCli, dynCli), nil
	}
	// Check if v1beta1 snapshot API exists
	exists, err = kube.IsGroupVersionAvailable(ctx, kubeCli.Discovery(), v1beta1.GroupName, v1beta1.Version)
	if err != nil {
		return nil, errors.Errorf("Failed to call discovery APIs: %v", err)
	}
	if exists {
		return NewSnapshotBeta(kubeCli, dynCli), nil
	}
	// Check if v1 (stable) snapshot API exists
	exists, err = kube.IsGroupVersionAvailable(ctx, kubeCli.Discovery(), GroupName, Version)
	if err != nil {
		return nil, errors.Errorf("Failed to call discovery APIs: %v", err)
	}
	if exists {
		return NewSnapshotStable(kubeCli, dynCli), nil
	}
	return nil, errors.New("Snapshot resources not supported")
}
