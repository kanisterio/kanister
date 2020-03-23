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
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/kube/snapshot/apis/v1alpha1"
	"github.com/kanisterio/kanister/pkg/poll"
)

const (
	pvcKind = "PersistentVolumeClaim"

	// Snapshot resource Kinds
	VolSnapClassKind   = "VolumeSnapshotClass"
	VolSnapKind        = "VolumeSnapshot"
	VolSnapContentKind = "VolumeSnapshotContent"
)

type SnapshotAlpha struct {
	kubeCli kubernetes.Interface
	dynCli  dynamic.Interface
}

func NewSnapshotAlpha(kubeCli kubernetes.Interface, dynCli dynamic.Interface) Snapshotter {
	return &SnapshotAlpha{kubeCli: kubeCli, dynCli: dynCli}
}

// GetVolumeSnapshotClass returns VolumeSnapshotClass name which is annotated with given key.
func (sna *SnapshotAlpha) GetVolumeSnapshotClass(annotationKey, annotationValue string) (string, error) {
	return getSnapshotClassbyAnnotation(sna.dynCli, v1alpha1.VolSnapClassGVR, annotationKey, annotationValue)
}

// Create creates a VolumeSnapshot and returns it or any error that happened meanwhile.
func (sna *SnapshotAlpha) Create(ctx context.Context, name, namespace, pvcName string, snapshotClass *string, waitForReady bool) error {
	if _, err := sna.kubeCli.CoreV1().PersistentVolumeClaims(namespace).Get(pvcName, metav1.GetOptions{}); err != nil {
		if k8errors.IsNotFound(err) {
			return errors.Errorf("Failed to find PVC %s, Namespace %s", pvcName, namespace)
		}
		return errors.Errorf("Failed to query PVC %s, Namespace %s: %v", pvcName, namespace, err)
	}

	snap := UnstructuredVolumeSnapshotAlpha(name, namespace, pvcName, "", *snapshotClass)
	_, err := sna.dynCli.Resource(v1alpha1.VolSnapGVR).Namespace(namespace).Create(snap, metav1.CreateOptions{})
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

// Get will return the VolumeSnapshot in the 'namespace' with given 'name'.
func (sna *SnapshotAlpha) Get(ctx context.Context, name, namespace string) (*v1alpha1.VolumeSnapshot, error) {
	us, err := sna.dynCli.Resource(v1alpha1.VolSnapGVR).Namespace(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	vs := &v1alpha1.VolumeSnapshot{}
	err = TransformUnstructured(us, vs)
	return vs, err
}

// Delete will delete the VolumeSnapshot and returns any error as a result.
func (sna *SnapshotAlpha) Delete(ctx context.Context, name, namespace string) error {
	if err := sna.dynCli.Resource(v1alpha1.VolSnapGVR).Namespace(namespace).Delete(name, &metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	// If the Snapshot does not exist, that's an acceptable error and we ignore it
	return nil
}

// Clone will clone the VolumeSnapshot to namespace 'cloneNamespace'.
// Underlying VolumeSnapshotContent will be cloned with a different name.
func (sna *SnapshotAlpha) Clone(ctx context.Context, name, namespace, cloneName, cloneNamespace string, waitForReady bool) error {
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
func (sna *SnapshotAlpha) GetSource(ctx context.Context, snapshotName, namespace string) (*Source, error) {
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
	content := UnstructuredVolumeSnapshotContentBeta(contentName, snapshotName, namespace, deletionPolicy, source.Driver, source.Handle, source.VolumeSnapshotClassName)
	snap := UnstructuredVolumeSnapshotAlpha(snapshotName, namespace, "", contentName, source.VolumeSnapshotClassName)
	_, err = sna.dynCli.Resource(v1alpha1.VolSnapContentGVR).Namespace("").Create(content, metav1.CreateOptions{})
	if err != nil {
		return errors.Errorf("Failed to create content, VolumesnapshotContent: %s, Error: %v", content.GetName(), err)
	}
	_, err = sna.dynCli.Resource(v1alpha1.VolSnapGVR).Namespace(namespace).Create(snap, metav1.CreateOptions{})
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
		us, err := sna.dynCli.Resource(v1alpha1.VolSnapGVR).Namespace(namespace).Get(snapshotName, metav1.GetOptions{})
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
	us, err := sna.dynCli.Resource(v1alpha1.VolSnapContentGVR).Namespace("").Get(contentName, metav1.GetOptions{})
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
	us, err := sna.dynCli.Resource(v1alpha1.VolSnapClassGVR).Namespace("").Get(snapClassName, metav1.GetOptions{})
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

func UnstructuredVolumeSnapshotAlpha(name, namespace, pvcName, contentName, snapClassName string) *unstructured.Unstructured {
	snap := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": fmt.Sprintf("%s/%s", v1alpha1.GroupName, v1alpha1.Version),
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
				"kind":      pvcKind,
				"name":      pvcName,
				"namespace": namespace,
			},
			"snapshotClassName": snapClassName,
			"deletionPolicy":    "Delete",
		}
	}
	if contentName != "" {
		snap.Object["spec"] = map[string]interface{}{
			"snapshotContentName": contentName,
			"snapshotClassName":   snapClassName,
			"deletionPolicy":      "Delete",
		}
	}
	return snap
}

func UnstructuredVolumeSnapshotContentAlpha(name, snapshotName, snapshotNs, deletionPolicy, driver, handle, snapClassName string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": fmt.Sprintf("%s/%s", v1alpha1.GroupName, v1alpha1.Version),
			"kind":       VolSnapContentKind,
			"metadata": map[string]interface{}{
				"name": name,
			},
			"spec": map[string]interface{}{
				"csiVolumeSnapshotSource": map[string]interface{}{
					"driver":         driver,
					"snapshotHandle": handle,
				},
				"volumeSnapshotRef": map[string]interface{}{
					"kind":      VolSnapKind,
					"name":      snapshotName,
					"namespace": snapshotNs,
				},
				"snapshotClassName": snapClassName,
				"deletionPolicy":    deletionPolicy,
			},
		},
	}
}

func UnstructuredVolumeSnapshotClassAlpha(name, driver, deletionPolicy string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": fmt.Sprintf("%s/%s", v1alpha1.GroupName, v1alpha1.Version),
			"kind":       VolSnapClassKind,
			"metadata": map[string]interface{}{
				"name": name,
			},
			"snapshotter":    driver,
			"deletionPolicy": deletionPolicy,
		},
	}
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

func getSnapshotClassbyAnnotation(dynCli dynamic.Interface, gvr schema.GroupVersionResource, annotationKey, annotationValue string) (string, error) {
	us, err := dynCli.Resource(gvr).Namespace("").List(metav1.ListOptions{})
	if err != nil {
		return "", errors.Errorf("Failed to get VolumeSnapshotClasses in the cluster: %v", err)
	}
	if us == nil || len(us.Items) == 0 {
		return "", errors.Errorf("Failed to find any VolumeSnapshotClass in the cluster: %v", err)
	}
	for _, vsc := range us.Items {
		ans := vsc.GetAnnotations()
		if val, ok := ans[annotationKey]; ok && val == annotationValue {
			return vsc.GetName(), nil
		}
	}
	return "", errors.Errorf("Failed to find VolumesnapshotClass with %s=%s annotation in the cluster", annotationKey, annotationValue)
}
