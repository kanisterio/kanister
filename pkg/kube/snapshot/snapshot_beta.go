// Copyright 2020 The Kanister Authors.
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
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	storage "k8s.io/api/storage/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/kube/snapshot/apis/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube/snapshot/apis/v1beta1"
	"github.com/kanisterio/kanister/pkg/poll"
)

type SnapshotBeta struct {
	kubeCli kubernetes.Interface
	dynCli  dynamic.Interface
}

func NewSnapshotBeta(kubeCli kubernetes.Interface, dynCli dynamic.Interface) Snapshotter {
	return &SnapshotBeta{kubeCli: kubeCli, dynCli: dynCli}
}

// GetVolumeSnapshotClass returns VolumeSnapshotClass name which is annotated with given key.
func (sna *SnapshotBeta) GetVolumeSnapshotClass(annotationKey, annotationValue, storageClassName string) (string, string, error) {
	scName, err := getSnapshotClassbyAnnotation(sna.dynCli, sna.kubeCli, v1beta1.VolSnapClassGVR, annotationKey, annotationValue, storageClassName)
	if err != nil {
		return "", "", err
	}
	deletionPolicy, err := sna.getDeletionPolicyFromClass(scName)
	if err != nil {
		return "", "", err
	}
	return scName, deletionPolicy, nil
}

// Create creates a VolumeSnapshot and returns it or any error happened meanwhile.
func (sna *SnapshotBeta) Create(ctx context.Context, name, namespace, volumeName string, snapshotClass *string, waitForReady bool) error {
	if _, err := sna.kubeCli.CoreV1().PersistentVolumeClaims(namespace).Get(volumeName, metav1.GetOptions{}); err != nil {
		if k8errors.IsNotFound(err) {
			return errors.Errorf("Failed to find PVC %s, Namespace %s", volumeName, namespace)
		}
		return errors.Wrapf(err, "Failed to query PVC %s, Namespace %s", volumeName, namespace)
	}

	snap := UnstructuredVolumeSnapshotBeta(name, namespace, volumeName, "", *snapshotClass)
	if _, err := sna.dynCli.Resource(v1beta1.VolSnapGVR).Namespace(namespace).Create(snap, metav1.CreateOptions{}); err != nil {
		return errors.Wrapf(err, "Failed to create snapshot resource %s, Namespace %s", name, namespace)
	}

	if !waitForReady {
		return nil
	}

	if err := sna.WaitOnReadyToUse(ctx, name, namespace); err != nil {
		return err
	}

	_, err := sna.Get(ctx, name, namespace)
	return err
}

// Get will return the VolumeSnapshot in the 'namespace' with given 'name'.
func (sna *SnapshotBeta) Get(ctx context.Context, name, namespace string) (*v1alpha1.VolumeSnapshot, error) {
	us, err := sna.dynCli.Resource(v1beta1.VolSnapGVR).Namespace(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	vs := &v1beta1.VolumeSnapshot{}
	if err := TransformUnstructured(us, vs); err != nil {
		return nil, err
	}
	// Populate v1alpha1.VolumeSnapshot object from v1beta1.VolumeSnapshot
	vsa := v1alpha1.VolumeSnapshot{}
	meta := vs.ObjectMeta.DeepCopy()
	if meta == nil {
		return nil, fmt.Errorf("Invalid VolumeSnapshotObject: ObjectMeta is nil")
	}
	vsa.ObjectMeta = *meta

	if vs.Spec.Source.PersistentVolumeClaimName != nil {
		vsa.Spec.Source = &corev1.TypedLocalObjectReference{
			Kind: PVCKind,
			Name: *vs.Spec.Source.PersistentVolumeClaimName,
		}
	}
	if vs.Spec.VolumeSnapshotClassName != nil {
		vsa.Spec.VolumeSnapshotClassName = *vs.Spec.VolumeSnapshotClassName
	}
	if vs.Spec.Source.VolumeSnapshotContentName != nil {
		vsa.Spec.SnapshotContentName = *vs.Spec.Source.VolumeSnapshotContentName
	}
	if vs.Status == nil {
		return &vsa, nil
	}
	// If Status is not nil, set VolumeSnapshotContentName from status
	vsa.Status = v1alpha1.VolumeSnapshotStatus{
		CreationTime: vs.Status.CreationTime,
		RestoreSize:  vs.Status.RestoreSize,
	}
	if vs.Status.BoundVolumeSnapshotContentName != nil {
		vsa.Spec.SnapshotContentName = *vs.Status.BoundVolumeSnapshotContentName
	}
	if vs.Status.ReadyToUse != nil {
		vsa.Status.ReadyToUse = *vs.Status.ReadyToUse
	}
	if vs.Status.Error != nil {
		vsa.Status.Error = &storage.VolumeError{
			Time:    *vs.Status.Error.Time,
			Message: *vs.Status.Error.Message,
		}
	}
	return &vsa, nil
}

// Delete will delete the VolumeSnapshot and returns any error as a result.
func (sna *SnapshotBeta) Delete(ctx context.Context, name, namespace string) error {
	snap, err := sna.Get(ctx, name, namespace)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "Failed to find VolumeSnapshot: %s/%s", namespace, name)
	}
	if err := sna.dynCli.Resource(v1beta1.VolSnapGVR).Namespace(namespace).Delete(name, &metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return errors.Wrapf(err, "Failed to delete VolumeSnapshot: %s/%s", namespace, name)
	}

	// If the Snapshot does not exist, that's an acceptable error and we ignore it

	return sna.DeleteContent(ctx, snap.Spec.SnapshotContentName)
}

// DeleteContent will delete the specified VolumeSnapshotContent
func (sna *SnapshotBeta) DeleteContent(ctx context.Context, name string) error {
	if err := sna.dynCli.Resource(v1beta1.VolSnapContentGVR).Delete(name, &metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return errors.Wrapf(err, "Failed to delete VolumeSnapshotContent: %s", name)
	}
	// If the Snapshot Content does not exist, that's an acceptable error and we ignore it
	return nil
}

// Clone will clone the VolumeSnapshot to namespace 'cloneNamespace'.
// Underlying VolumeSnapshotContent will be cloned with a different name.
func (sna *SnapshotBeta) Clone(ctx context.Context, name, namespace, cloneName, cloneNamespace string, waitForReady bool) error {
	_, err := sna.Get(ctx, cloneName, cloneNamespace)
	if err == nil {
		return errors.Errorf("Target snapshot already exists in target namespace, Volumesnapshot: %s, Namespace: %s", cloneName, cloneNamespace)
	}
	if !k8errors.IsNotFound(err) {
		return errors.Errorf("Failed to query target Volumesnapshot: %s, Namespace: %s: %v", cloneName, cloneNamespace, err)
	}

	src, err := sna.GetSource(ctx, name, namespace)
	if err != nil {
		return errors.Errorf("Failed to get source")
	}
	return sna.CreateFromSource(ctx, src, cloneName, cloneNamespace, waitForReady)
}

// GetSource will return the CSI source that backs the volume snapshot.
func (sna *SnapshotBeta) GetSource(ctx context.Context, snapshotName, namespace string) (*Source, error) {
	snap, err := sna.Get(ctx, snapshotName, namespace)
	if err != nil {
		return nil, errors.Errorf("Failed to get snapshot, VolumeSnapshot: %s, Error: %v", snapshotName, err)
	}
	if !snap.Status.ReadyToUse {
		return nil, errors.Errorf("Snapshot is not ready, VolumeSnapshot: %s, Namespace: %s", snapshotName, namespace)
	}
	if snap.Spec.SnapshotContentName == "" {
		return nil, errors.Errorf("Snapshot does not have content, VolumeSnapshot: %s, Namespace: %s", snapshotName, namespace)
	}

	cont, err := sna.getContent(ctx, snap.Spec.SnapshotContentName)
	if err != nil {
		return nil, errors.Errorf("Failed to get snapshot content, VolumeSnapshot: %s, VolumeSnapshotContent: %s, Error: %v", snapshotName, snap.Spec.SnapshotContentName, err)
	}
	src := &Source{
		Handle:                  *cont.Status.SnapshotHandle,
		Driver:                  cont.Spec.Driver,
		RestoreSize:             cont.Status.RestoreSize,
		VolumeSnapshotClassName: *cont.Spec.VolumeSnapshotClassName,
	}
	return src, nil
}

// CreateFromSource will create a 'Volumesnapshot' and 'VolumesnaphotContent' pair for the underlying snapshot source.
func (sna *SnapshotBeta) CreateFromSource(ctx context.Context, source *Source, snapshotName, namespace string, waitForReady bool) error {
	deletionPolicy, err := sna.getDeletionPolicyFromClass(source.VolumeSnapshotClassName)
	if err != nil {
		return errors.Wrap(err, "Failed to get DeletionPolicy from VolumeSnapshotClass")
	}
	contentName := snapshotName + "-content-" + string(uuid.NewUUID())
	snap := UnstructuredVolumeSnapshotBeta(snapshotName, namespace, "", contentName, source.VolumeSnapshotClassName)

	if err := sna.CreateContentFromSource(ctx, source, contentName, snapshotName, namespace, deletionPolicy); err != nil {
		return err
	}
	if _, err := sna.dynCli.Resource(v1beta1.VolSnapGVR).Namespace(namespace).Create(snap, metav1.CreateOptions{}); err != nil {
		return errors.Errorf("Failed to create content, Volumesnapshot: %s, Error: %v", snap.GetName(), err)
	}
	if !waitForReady {
		return nil
	}

	err = sna.WaitOnReadyToUse(ctx, snapshotName, namespace)
	return err
}

// CreateContentFromSource will create a 'VolumesnaphotContent' for the underlying snapshot source.
func (sna *SnapshotBeta) CreateContentFromSource(ctx context.Context, source *Source, contentName, snapshotName, namespace, deletionPolicy string) error {
	content := UnstructuredVolumeSnapshotContentBeta(contentName, snapshotName, namespace, deletionPolicy, source.Driver, source.Handle, source.VolumeSnapshotClassName)
	if _, err := sna.dynCli.Resource(v1beta1.VolSnapContentGVR).Create(content, metav1.CreateOptions{}); err != nil {
		return errors.Errorf("Failed to create content, VolumesnapshotContent: %s, Error: %v", content.GetName(), err)
	}
	return nil
}

// WaitOnReadyToUse will block until the Volumesnapshot in 'namespace' with name 'snapshotName'
// has status 'ReadyToUse' or 'ctx.Done()' is signalled.
func (sna *SnapshotBeta) WaitOnReadyToUse(ctx context.Context, snapshotName, namespace string) error {
	return poll.Wait(ctx, func(context.Context) (bool, error) {
		us, err := sna.dynCli.Resource(v1beta1.VolSnapGVR).Namespace(namespace).Get(snapshotName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		vs := v1beta1.VolumeSnapshot{}
		err = TransformUnstructured(us, &vs)
		if err != nil {
			return false, err
		}
		if vs.Status == nil {
			return false, nil
		}
		// Error can be set while waiting for creation
		if vs.Status.Error != nil {
			return false, errors.New(*vs.Status.Error.Message)
		}
		return (vs.Status.ReadyToUse != nil && *vs.Status.ReadyToUse && vs.Status.CreationTime != nil), nil
	})
}

func (sna *SnapshotBeta) getContent(ctx context.Context, contentName string) (*v1beta1.VolumeSnapshotContent, error) {
	us, err := sna.dynCli.Resource(v1beta1.VolSnapContentGVR).Get(contentName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	vsc := v1beta1.VolumeSnapshotContent{}
	if err := TransformUnstructured(us, &vsc); err != nil {
		return nil, err
	}
	return &vsc, nil
}

func (sna *SnapshotBeta) getDeletionPolicyFromClass(snapClassName string) (string, error) {
	us, err := sna.dynCli.Resource(v1beta1.VolSnapClassGVR).Get(snapClassName, metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrapf(err, "Failed to find VolumeSnapshotClass: %s", snapClassName)
	}
	vsc := v1beta1.VolumeSnapshotClass{}
	if err := TransformUnstructured(us, &vsc); err != nil {
		return "", err
	}
	return vsc.DeletionPolicy, nil
}

func UnstructuredVolumeSnapshotBeta(name, namespace, pvcName, contentName, snapClassName string) *unstructured.Unstructured {
	snap := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": fmt.Sprintf("%s/%s", v1beta1.GroupName, v1beta1.Version),
			"kind":       VolSnapKind,
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
		},
	}
	if pvcName != "" {
		snap.Object["spec"] = map[string]interface{}{
			"source": map[string]interface{}{
				"persistentVolumeClaimName": pvcName,
			},
			"volumeSnapshotClassName": snapClassName,
		}
	}
	if contentName != "" {
		snap.Object["spec"] = map[string]interface{}{
			"source": map[string]interface{}{
				"volumeSnapshotContentName": contentName,
			},
			"volumeSnapshotClassName": snapClassName,
		}
	}
	return snap
}

func UnstructuredVolumeSnapshotContentBeta(name, snapshotName, snapshotNs, deletionPolicy, driver, handle, snapClassName string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": fmt.Sprintf("%s/%s", v1beta1.GroupName, v1beta1.Version),
			"kind":       VolSnapContentKind,
			"metadata": map[string]interface{}{
				"name": name,
			},
			"spec": map[string]interface{}{
				"volumeSnapshotRef": map[string]interface{}{
					"kind":      VolSnapKind,
					"name":      snapshotName,
					"namespace": snapshotNs,
				},
				"deletionPolicy": deletionPolicy,
				"driver":         driver,
				"source": map[string]interface{}{
					"snapshotHandle": handle,
				},
				"volumeSnapshotClassName": snapClassName,
			},
		},
	}
}

func UnstructuredVolumeSnapshotClassBeta(name, driver, deletionPolicy string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": fmt.Sprintf("%s/%s", v1beta1.GroupName, v1beta1.Version),
			"kind":       VolSnapClassKind,
			"metadata": map[string]interface{}{
				"name": name,
			},
			VolSnapClassBetaDriverKey: driver,
			"deletionPolicy":          deletionPolicy,
		},
	}
}
