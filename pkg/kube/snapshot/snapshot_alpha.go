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
	"encoding/json"
	"fmt"

	"github.com/kanisterio/kanister/pkg/kube/snapshot/apis/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/poll"
)

const (
	pvcKind = "PersistentVolumeClaim"

	VersionAlpha = "v1alpha1"
	Group        = "snapshot.storage.k8s.io"

	// Snapshot resource Kinds
	VolSnapClassKind   = "VolumeSnapshotClass"
	VolSnapKind        = "VolumeSnapshot"
	VolSnapContentKind = "VolumeSnapshotContent"

	volSnapClassResource   = "volumesnapshotclasses"
	volSnapResource        = "volumesnapshots"
	volSnapContentResource = "volumesnapshotcontents"
)

var (
	// VolSnapGVR specifies GVR schema for VolumeSnapshots
	VolSnapGVR = schema.GroupVersionResource{Group: Group, Version: VersionAlpha, Resource: volSnapResource}
	// VolSnapClassGVR specifies GVR schema for VolumeSnapshotClasses
	VolSnapClassGVR = schema.GroupVersionResource{Group: Group, Version: VersionAlpha, Resource: volSnapClassResource}
	// VolSnapContentGVR specifies GVR schema for VolumeSnapshotContents
	VolSnapContentGVR = schema.GroupVersionResource{Group: Group, Version: VersionAlpha, Resource: volSnapContentResource}
)

type SnapshotAlpha struct {
	dynCli  dynamic.Interface
	kubeCli kubernetes.Interface
}

func NewSnapshotAlpha(dynCli dynamic.Interface, kubeCli kubernetes.Interface) Snapshotter {
	return &SnapshotAlpha{
		dynCli:  dynCli,
		kubeCli: kubeCli,
	}
}

// Create creates a VolumeSnapshot and returns it or any error happened meanwhile.
func (sna *SnapshotAlpha) Create(ctx context.Context, name, namespace, volumeName string, snapshotClass *string, waitForReady bool) error {
	if _, err := sna.kubeCli.CoreV1().PersistentVolumeClaims(namespace).Get(volumeName, metav1.GetOptions{}); err != nil {
		if k8errors.IsNotFound(err) {
			return errors.Errorf("Failed to find PVC %s, Namespace %s", volumeName, namespace)
		}
		return errors.Errorf("Failed to query PVC %s, Namespace %s: %v", volumeName, namespace, err)
	}

	err := sna.createVolumeSnapshot(name, namespace, corev1.ObjectReference{Kind: pvcKind, Name: volumeName}, *snapshotClass)
	if err != nil {
		return err
	}

	if !waitForReady {
		return nil
	}

	err = sna.WaitOnReadyToUse(ctx, name, namespace)
	if err != nil {
		return err
	}

	_, err = sna.Get(ctx, name, namespace)
	return err
}

// Get will return the VolumeSnapshot in the namespace 'namespace' with given 'name'.
func (sna *SnapshotAlpha) Get(ctx context.Context, name, namespace string) (*v1alpha1.VolumeSnapshot, error) {
	us, err := sna.dynCli.Resource(VolSnapGVR).Namespace(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	vs := &v1alpha1.VolumeSnapshot{}
	err = TransformUnstructured(us, vs)
	return vs, err
}

// Delete will delete the VolumeSnapshot and returns any error as a result.
func (sna *SnapshotAlpha) Delete(ctx context.Context, name, namespace string) error {
	if err := sna.dynCli.Resource(VolSnapGVR).Namespace(namespace).Delete(name, &metav1.DeleteOptions{}); !apierrors.IsNotFound(err) {
		return err
	}

	// If the Snapshot does not exist, that's an acceptable error and we ignore it
	return nil
}

// Clone will clone the VolumeSnapshot to namespace 'cloneNamespace'.
// Underlying VolumeSnapshotContent will be cloned with a different name.
func (sna *SnapshotAlpha) Clone(ctx context.Context, name, namespace, cloneName, cloneNamespace string, waitForReady bool) error {
	snap, err := sna.Get(ctx, name, namespace)
	if err != nil {
		return err
	}
	if !snap.Status.ReadyToUse {
		return errors.Errorf("Original snapshot is not ready, VolumeSnapshot: %s, Namespace: %s", cloneName, cloneNamespace)
	}
	if snap.Spec.SnapshotContentName == "" {
		return errors.Errorf("Original snapshot does not have content, VolumeSnapshot: %s, Namespace: %s", cloneName, cloneNamespace)
	}

	_, err = sna.Get(ctx, cloneName, cloneNamespace)
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
func (sna *SnapshotAlpha) GetSource(ctx context.Context, snapshotName, namespace string) (*Source, error) {
	snap, err := sna.Get(ctx, snapshotName, namespace)
	if err != nil {
		return nil, errors.Errorf("Failed to get snapshot, VolumeSnapshot: %s, Error: %v", snapshotName, err)
	}
	cont, err := sna.getContent(ctx, snap.Spec.SnapshotContentName)
	if err != nil {
		return nil, errors.Errorf("Failed to get snapshot content, VolumeSnapshot: %s, VolumeSnapshotContent: %s, Error: %v", snapshotName, snap.Spec.SnapshotContentName, err)
	}
	src := &Source{
		Handle:                  cont.Spec.CSI.SnapshotHandle,
		Driver:                  cont.Spec.CSI.Driver,
		RestoreSize:             cont.Spec.CSI.RestoreSize,
		VolumeSnapshotClassName: cont.Spec.VolumeSnapshotClassName,
	}
	return src, nil
}

// CreateFromSource will create a 'Volumesnapshot' and 'VolumesnaphotContent' pair for the underlying snapshot source.
func (sna *SnapshotAlpha) CreateFromSource(ctx context.Context, source *Source, snapshotName, namespace string, waitForReady bool) error {
	deletionPolicy, err := sna.getDeletionPolicyFromClass(source.VolumeSnapshotClassName)
	if err != nil {
		return errors.Wrap(err, "Failed to get DeletionPolicy from VolumeSnapshotClass")
	}
	contentName := snapshotName + "-content-" + string(uuid.NewUUID())

	content := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": fmt.Sprintf("%s/%s", Group, VersionAlpha),
			"kind":       VolSnapContentKind,
			"metadata": map[string]interface{}{
				"name": contentName,
			},
			"spec": map[string]interface{}{
				"csiVolumeSnapshotSource": map[string]interface{}{
					"driver":         source.Driver,
					"snapshotHandle": source.Handle,
				},
				"volumeSnapshotRef": map[string]interface{}{
					"kind":      VolSnapKind,
					"name":      snapshotName,
					"namespace": snapshotName,
				},
				"snapshotClassName": source.VolumeSnapshotClassName,
				"deletionPolicy":    deletionPolicy,
			},
		},
	}

	snap := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": fmt.Sprintf("%s/%s", Group, VersionAlpha),
			"kind":       VolSnapKind,
			"metadata": map[string]interface{}{
				"name": snapshotName,
			},
			"spec": map[string]interface{}{
				"snapshotContentName": contentName,
				"snapshotClassName":   source.VolumeSnapshotClassName,
			},
		},
	}

	_, err = sna.dynCli.Resource(VolSnapContentGVR).Namespace("").Create(content, metav1.CreateOptions{})
	if err != nil {
		return errors.Errorf("Failed to create content, VolumesnapshotContent: %s, Error: %v", content.GetName(), err)
	}
	_, err = sna.dynCli.Resource(VolSnapGVR).Namespace(namespace).Create(snap, metav1.CreateOptions{})
	if err != nil {
		return errors.Errorf("Failed to create content, Volumesnapshot: %s, Error: %v", snap.GetName(), err)
	}
	if !waitForReady {
		return nil
	}

	err = sna.WaitOnReadyToUse(ctx, snapshotName, namespace)
	return err
}

// WaitOnReadyToUse will block until the Volumesnapshot in namespace 'namespace' with name 'snapshotName'
// has status 'ReadyToUse' or 'ctx.Done()' is signalled.
func (sna *SnapshotAlpha) WaitOnReadyToUse(ctx context.Context, snapshotName, namespace string) error {
	return poll.Wait(ctx, func(context.Context) (bool, error) {
		us, err := sna.dynCli.Resource(VolSnapGVR).Namespace(namespace).Get(snapshotName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		vs := v1alpha1.VolumeSnapshot{}
		err = TransformUnstructured(us, &vs)
		if err != nil {
			return false, err
		}
		// Error can be set while waiting for creation
		if vs.Status.Error != nil {
			return false, errors.New(vs.Status.Error.Message)
		}
		return (vs.Status.ReadyToUse && vs.Status.CreationTime != nil), nil
	})
}

func (sna *SnapshotAlpha) getContent(ctx context.Context, contentName string) (*v1alpha1.VolumeSnapshotContent, error) {
	us, err := sna.dynCli.Resource(VolSnapContentGVR).Namespace("").Get(contentName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	vsc := v1alpha1.VolumeSnapshotContent{}
	err = TransformUnstructured(us, &vsc)
	if err != nil {
		return nil, err
	}
	return &vsc, nil
}

func (sna *SnapshotAlpha) getDeletionPolicyFromClass(snapClassName string) (string, error) {
	us, err := sna.dynCli.Resource(VolSnapClassGVR).Namespace("").Get(snapClassName, metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrapf(err, "Failed to find VolumeSnapshotClass: %s", snapClassName)
	}
	vsc := v1alpha1.VolumeSnapshotClass{}
	err = TransformUnstructured(us, &vsc)
	if err != nil {
		return "", err
	}
	return vsc.DeletionPolicy, nil
}

func (sna *SnapshotAlpha) createVolumeSnapshot(name, namespace string, pvcObjectRef corev1.ObjectReference, snapClassName string) error {
	snap := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": fmt.Sprintf("%s/%s", Group, VersionAlpha),
			"kind":       VolSnapKind,
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"source": map[string]interface{}{
					"kind":      pvcObjectRef.Kind,
					"name":      pvcObjectRef.Name,
					"namespace": pvcObjectRef.Namespace,
				},
				"snapshotClassName": snapClassName,
			},
		},
	}

	_, err := sna.dynCli.Resource(VolSnapGVR).Namespace(namespace).Create(snap, metav1.CreateOptions{})
	return err
}

// TransformUnstructured maps Unstructured object to object pointed by value
func TransformUnstructured(u *unstructured.Unstructured, value interface{}) error {
	b, err := json.Marshal(u.Object)
	if err != nil {
		return errors.Errorf("Failed to Marshal unstructured object: %v", err)
	}
	err = json.Unmarshal(b, value)
	if err != nil {
		return errors.Errorf("Failed to Unmarshal unstructured object: %v", err)
	}
	return nil
}
