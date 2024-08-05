// Copyright 2021 The Kanister Authors.
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

	"github.com/kanisterio/errkit"
	v1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/blockstorage"
)

const (
	// VolumeSnapshotContentResourcePlural is "volumesnapshotcontents"
	VolumeSnapshotContentResourcePlural = "volumesnapshotcontents"
	// VolumeSnapshotResourcePlural is "volumesnapshots"
	VolumeSnapshotResourcePlural = "volumesnapshots"
	// VolumeSnapshotClassResourcePlural is "volumesnapshotclasses"
	VolumeSnapshotClassResourcePlural = "volumesnapshotclasses"

	GroupName = "snapshot.storage.k8s.io"
	Version   = "v1"
)

var (
	// VolSnapGVR specifies GVR schema for VolumeSnapshots
	VolSnapGVR = schema.GroupVersionResource{Group: GroupName, Version: Version, Resource: VolumeSnapshotResourcePlural}
	// VolSnapClassGVR specifies GVR schema for VolumeSnapshotClasses
	VolSnapClassGVR = schema.GroupVersionResource{Group: GroupName, Version: Version, Resource: VolumeSnapshotClassResourcePlural}
	// VolSnapContentGVR specifies GVR schema for VolumeSnapshotContents
	VolSnapContentGVR = schema.GroupVersionResource{Group: GroupName, Version: Version, Resource: VolumeSnapshotContentResourcePlural}
)

type SnapshotStable struct {
	kubeCli kubernetes.Interface
	dynCli  dynamic.Interface
}

func NewSnapshotStable(kubeCli kubernetes.Interface, dynCli dynamic.Interface) Snapshotter {
	return &SnapshotStable{kubeCli: kubeCli, dynCli: dynCli}
}

// CloneVolumeSnapshotClass creates a copy of the source volume snapshot class
func (sna *SnapshotStable) CloneVolumeSnapshotClass(ctx context.Context, sourceClassName, targetClassName, newDeletionPolicy string, excludeAnnotations []string) error {
	return cloneSnapshotClass(ctx, sna.dynCli, VolSnapClassGVR, sourceClassName, targetClassName, newDeletionPolicy, excludeAnnotations)
}

// GetVolumeSnapshotClass returns VolumeSnapshotClass name which is annotated with given key.
func (sna *SnapshotStable) GetVolumeSnapshotClass(ctx context.Context, annotationKey, annotationValue, storageClassName string) (string, error) {
	return GetSnapshotClassbyAnnotation(ctx, sna.dynCli, sna.kubeCli, VolSnapClassGVR, annotationKey, annotationValue, storageClassName)
}

// Create creates a VolumeSnapshot and returns it or any error happened meanwhile.
func (sna *SnapshotStable) Create(ctx context.Context, volumeName string, snapshotClass *string, waitForReady bool, snapshotMeta ObjectMeta) error {
	return createSnapshot(ctx, sna.dynCli, sna.kubeCli, VolSnapGVR, volumeName, snapshotClass, waitForReady, snapshotMeta)
}

// Get will return the VolumeSnapshot in the 'namespace' with given 'name'.
func (sna *SnapshotStable) Get(ctx context.Context, name, namespace string) (*v1.VolumeSnapshot, error) {
	return getSnapshot(ctx, sna.dynCli, VolSnapGVR, name, namespace)
}

func (sna *SnapshotStable) List(ctx context.Context, namespace string, labels map[string]string) (*v1.VolumeSnapshotList, error) {
	return listSnapshots(ctx, sna.dynCli, VolSnapGVR, namespace, labels)
}

// Delete will delete the VolumeSnapshot and returns any error as a result.
func (sna *SnapshotStable) Delete(ctx context.Context, name, namespace string) (*v1.VolumeSnapshot, error) {
	return deleteSnapshot(ctx, sna.dynCli, VolSnapGVR, name, namespace)
}

// DeleteContent will delete the specified VolumeSnapshotContent
func (sna *SnapshotStable) DeleteContent(ctx context.Context, name string) error {
	if err := sna.dynCli.Resource(VolSnapContentGVR).Delete(ctx, name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return errkit.Wrap(err, "Failed to delete", "volumeSnapshotContent", name)
	}
	// If the Snapshot Content does not exist, that's an acceptable error and we ignore it
	return nil
}

// Clone will clone the VolumeSnapshot to namespace 'cloneNamespace'.
// Underlying VolumeSnapshotContent will be cloned with a different name.
func (sna *SnapshotStable) Clone(ctx context.Context, name, namespace string, waitForReady bool, snapshotMeta, snapshotContentMeta ObjectMeta) error {
	_, err := sna.Get(ctx, snapshotMeta.Name, snapshotMeta.Namespace)
	if err == nil {
		return errkit.New("Target snapshot already exists in target namespace", "volumeSnapshot", snapshotMeta.Name, "namespace", snapshotMeta.Namespace)
	}
	if !apierrors.IsNotFound(err) {
		return errkit.Wrap(err, "Failed to query target", "volumeSnapshot", snapshotMeta.Name, "namespace", snapshotMeta.Namespace)
	}

	src, err := sna.GetSource(ctx, name, namespace)
	if err != nil {
		return errkit.New("Failed to get source")
	}
	return sna.CreateFromSource(ctx, src, waitForReady, snapshotMeta, snapshotContentMeta)
}

// GetSource will return the CSI source that backs the volume snapshot.
func (sna *SnapshotStable) GetSource(ctx context.Context, snapshotName, namespace string) (*Source, error) {
	return getSnapshotSource(ctx, sna.dynCli, VolSnapGVR, VolSnapContentGVR, snapshotName, namespace)
}

// CreateFromSource will create a 'Volumesnapshot' and 'VolumesnaphotContent' pair for the underlying snapshot source.
func (sna *SnapshotStable) CreateFromSource(ctx context.Context, source *Source, waitForReady bool, snapshotMeta, snapshotContentMeta ObjectMeta) error {
	deletionPolicy, err := getDeletionPolicyFromClass(sna.dynCli, VolSnapClassGVR, source.VolumeSnapshotClassName)
	if err != nil {
		return errkit.Wrap(err, "Failed to get DeletionPolicy from VolumeSnapshotClass")
	}
	snapshotContentMeta.Name = snapshotMeta.Name + "-content-" + string(uuid.NewUUID())
	snapshotMeta.Labels = blockstorage.SanitizeTags(snapshotMeta.Labels)
	snap := UnstructuredVolumeSnapshot(
		VolSnapGVR,
		"",
		source.VolumeSnapshotClassName,
		snapshotMeta,
		snapshotContentMeta,
	)

	if err := sna.CreateContentFromSource(ctx, source, snapshotMeta.Name, snapshotMeta.Namespace, deletionPolicy, snapshotContentMeta); err != nil {
		return err
	}
	if _, err := sna.dynCli.Resource(VolSnapGVR).Namespace(snapshotMeta.Namespace).Create(ctx, snap, metav1.CreateOptions{}); err != nil {
		return errkit.Wrap(err, "Failed to create content", "volumeSnapshot", snap.GetName())
	}
	if !waitForReady {
		return nil
	}
	err = sna.WaitOnReadyToUse(ctx, snapshotMeta.Name, snapshotMeta.Namespace)
	return err
}

// UpdateVolumeSnapshotStatusStable sets the readyToUse valuse of a VolumeSnapshot.
func (sna *SnapshotStable) UpdateVolumeSnapshotStatusStable(ctx context.Context, namespace string, snapshotName string, readyToUse bool) error {
	return updateVolumeSnapshotStatus(ctx, sna.dynCli, VolSnapGVR, namespace, snapshotName, readyToUse)
}

// CreateContentFromSource will create a 'VolumesnaphotContent' for the underlying snapshot source.
func (sna *SnapshotStable) CreateContentFromSource(ctx context.Context, source *Source, snapshotName, snapshotNS, deletionPolicy string, snapshotContentMeta ObjectMeta) error {
	content := UnstructuredVolumeSnapshotContent(VolSnapContentGVR, snapshotName, snapshotNS, deletionPolicy, source.Driver, source.Handle, source.VolumeSnapshotClassName, snapshotContentMeta)
	if _, err := sna.dynCli.Resource(VolSnapContentGVR).Create(ctx, content, metav1.CreateOptions{}); err != nil {
		return errkit.Wrap(err, "Failed to create content", "volumeSnapshotContent", content.GetName())
	}
	return nil
}

// WaitOnReadyToUse will block until the Volumesnapshot in 'namespace' with name 'snapshotName'
// has status 'ReadyToUse' or 'ctx.Done()' is signalled.
func (sna *SnapshotStable) WaitOnReadyToUse(ctx context.Context, snapshotName, namespace string) error {
	return waitOnReadyToUse(ctx, sna.dynCli, VolSnapGVR, snapshotName, namespace, isReadyToUseBeta)
}

func (sna *SnapshotStable) GroupVersion(ctx context.Context) schema.GroupVersion {
	return schema.GroupVersion{
		Group:   GroupName,
		Version: Version,
	}
}
