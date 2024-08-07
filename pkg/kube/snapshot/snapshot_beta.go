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

	"github.com/kanisterio/errkit"
	v1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	pkglabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/blockstorage"
	"github.com/kanisterio/kanister/pkg/kube/snapshot/apis/v1beta1"
)

type SnapshotBeta struct {
	kubeCli kubernetes.Interface
	dynCli  dynamic.Interface
}

func NewSnapshotBeta(kubeCli kubernetes.Interface, dynCli dynamic.Interface) Snapshotter {
	return &SnapshotBeta{kubeCli: kubeCli, dynCli: dynCli}
}

// CloneVolumeSnapshotClass creates a copy of the source volume snapshot class
func (sna *SnapshotBeta) CloneVolumeSnapshotClass(ctx context.Context, sourceClassName, targetClassName, newDeletionPolicy string, excludeAnnotations []string) error {
	return cloneSnapshotClass(ctx, sna.dynCli, v1beta1.VolSnapClassGVR, sourceClassName, targetClassName, newDeletionPolicy, excludeAnnotations)
}

func cloneSnapshotClass(ctx context.Context, dynCli dynamic.Interface, snapClassGVR schema.GroupVersionResource, sourceClassName, targetClassName, newDeletionPolicy string, excludeAnnotations []string) error {
	usSourceSnapClass, err := dynCli.Resource(snapClassGVR).Get(ctx, sourceClassName, metav1.GetOptions{})
	if err != nil {
		return errkit.Wrap(err, "Failed to find source VolumeSnapshotClass", "volumeSnapshotClass", sourceClassName)
	}

	sourceSnapClass := v1.VolumeSnapshotClass{}
	if err := TransformUnstructured(usSourceSnapClass, &sourceSnapClass); err != nil {
		return err
	}
	existingAnnotations := sourceSnapClass.GetAnnotations()
	for _, key := range excludeAnnotations {
		delete(existingAnnotations, key)
	}
	usNew := UnstructuredVolumeSnapshotClass(snapClassGVR, targetClassName, sourceSnapClass.Driver, newDeletionPolicy, sourceSnapClass.Parameters)
	// Set Annotations/Labels
	usNew.SetAnnotations(existingAnnotations)
	usNew.SetLabels(map[string]string{CloneVolumeSnapshotClassLabelName: sourceClassName})
	if _, err = dynCli.Resource(snapClassGVR).Create(ctx, usNew, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		return errkit.Wrap(err, "Failed to create VolumeSnapshotClass", "volumeSnapshotClass", targetClassName)
	}
	return nil
}

// GetVolumeSnapshotClass returns VolumeSnapshotClass name which is annotated with given key.
func (sna *SnapshotBeta) GetVolumeSnapshotClass(ctx context.Context, annotationKey, annotationValue, storageClassName string) (string, error) {
	return GetSnapshotClassbyAnnotation(ctx, sna.dynCli, sna.kubeCli, v1beta1.VolSnapClassGVR, annotationKey, annotationValue, storageClassName)
}

// Create creates a VolumeSnapshot and returns it or any error happened meanwhile.
func (sna *SnapshotBeta) Create(ctx context.Context, volumeName string, snapshotClass *string, waitForReady bool, snapshotMeta ObjectMeta) error {
	return createSnapshot(ctx, sna.dynCli, sna.kubeCli, v1beta1.VolSnapGVR, volumeName, snapshotClass, waitForReady, snapshotMeta)
}

func createSnapshot(
	ctx context.Context,
	dynCli dynamic.Interface,
	kubeCli kubernetes.Interface,
	snapGVR schema.GroupVersionResource,
	volumeName string,
	snapshotClass *string,
	waitForReady bool,
	snapshotMeta ObjectMeta,
) error {
	if _, err := kubeCli.CoreV1().PersistentVolumeClaims(snapshotMeta.Namespace).Get(ctx, volumeName, metav1.GetOptions{}); err != nil {
		if apierrors.IsNotFound(err) {
			return errkit.New("Failed to find PVC", "pvc", volumeName, "namespace", snapshotMeta.Namespace)
		}
		return errkit.Wrap(err, "Failed to query PVC", "pvc", volumeName, "namespace", snapshotMeta.Namespace)
	}
	snapshotMeta.Labels = blockstorage.SanitizeTags(snapshotMeta.Labels)
	snapshotContentMeta := ObjectMeta{}
	snap := UnstructuredVolumeSnapshot(snapGVR, volumeName, *snapshotClass, snapshotMeta, snapshotContentMeta)
	if _, err := dynCli.Resource(snapGVR).Namespace(snapshotMeta.Namespace).Create(ctx, snap, metav1.CreateOptions{}); err != nil {
		return errkit.Wrap(err, "Failed to create snapshot resource", "name", snapshotMeta.Name, "namespace", snapshotMeta.Namespace)
	}

	if !waitForReady {
		return nil
	}

	if err := waitOnReadyToUse(ctx, dynCli, snapGVR, snapshotMeta.Name, snapshotMeta.Namespace, isReadyToUseBeta); err != nil {
		return err
	}

	_, err := getSnapshot(ctx, dynCli, snapGVR, snapshotMeta.Name, snapshotMeta.Namespace)
	return err
}

// Get will return the VolumeSnapshot in the 'namespace' with given 'name'.
func (sna *SnapshotBeta) Get(ctx context.Context, name, namespace string) (*v1.VolumeSnapshot, error) {
	return getSnapshot(ctx, sna.dynCli, v1beta1.VolSnapGVR, name, namespace)
}

func getSnapshot(ctx context.Context, dynCli dynamic.Interface, snapGVR schema.GroupVersionResource, name, namespace string) (*v1.VolumeSnapshot, error) {
	us, err := dynCli.Resource(snapGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	vs := &v1.VolumeSnapshot{}
	if err := TransformUnstructured(us, vs); err != nil {
		return nil, err
	}
	return vs, nil
}

func (sna *SnapshotBeta) List(ctx context.Context, namespace string, labels map[string]string) (*v1.VolumeSnapshotList, error) {
	return listSnapshots(ctx, sna.dynCli, v1beta1.VolSnapGVR, namespace, labels)
}

func listSnapshots(ctx context.Context, dynCli dynamic.Interface, snapGVR schema.GroupVersionResource, namespace string, labels map[string]string) (*v1.VolumeSnapshotList, error) {
	listOptions := metav1.ListOptions{}
	if labels != nil {
		labelSelector := metav1.LabelSelector{MatchLabels: blockstorage.SanitizeTags(labels)}
		listOptions.LabelSelector = pkglabels.Set(labelSelector.MatchLabels).String()
	}
	usList, err := dynCli.Resource(snapGVR).Namespace(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, err
	}
	vsList := &v1.VolumeSnapshotList{}
	for _, us := range usList.Items {
		vs := &v1.VolumeSnapshot{}
		if err := TransformUnstructured(&us, vs); err != nil {
			return nil, err
		}
		vsList.Items = append(vsList.Items, *vs)
	}
	return vsList, nil
}

// Delete will delete the VolumeSnapshot and returns any error as a result.
func (sna *SnapshotBeta) Delete(ctx context.Context, name, namespace string) (*v1.VolumeSnapshot, error) {
	return deleteSnapshot(ctx, sna.dynCli, v1beta1.VolSnapGVR, name, namespace)
}

func deleteSnapshot(ctx context.Context, dynCli dynamic.Interface, snapGVR schema.GroupVersionResource, name, namespace string) (*v1.VolumeSnapshot, error) {
	snap, err := getSnapshot(ctx, dynCli, snapGVR, name, namespace)
	if apierrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to find VolumeSnapshot", "namespace", namespace, "name", name)
	}
	if err := dynCli.Resource(snapGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return nil, errkit.Wrap(err, "Failed to delete VolumeSnapshot", "namespace", namespace, "name", name)
	}
	// If the Snapshot does not exist, that's an acceptable error and we ignore it
	return snap, nil
}

// DeleteContent will delete the specified VolumeSnapshotContent
func (sna *SnapshotBeta) DeleteContent(ctx context.Context, name string) error {
	if err := sna.dynCli.Resource(v1beta1.VolSnapContentGVR).Delete(ctx, name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return errkit.Wrap(err, "Failed to delete VolumeSnapshotContent", "name", name)
	}
	// If the Snapshot Content does not exist, that's an acceptable error and we ignore it
	return nil
}

// Clone will clone the VolumeSnapshot to namespace 'snapshotMeta.Namespace'.
// Underlying VolumeSnapshotContent will be cloned with a different name.
func (sna *SnapshotBeta) Clone(ctx context.Context, name, namespace string, waitForReady bool, snapshotMeta, snapshotContentMeta ObjectMeta) error {
	_, err := sna.Get(ctx, snapshotMeta.Name, snapshotMeta.Namespace)
	if err == nil {
		return errkit.New("Target snapshot already exists in target namespace", "volumeSnapshot", snapshotMeta.Name, "namespace", snapshotMeta.Namespace)
	}
	if !apierrors.IsNotFound(err) {
		return errkit.Wrap(err, "Failed to query target Volumesnapshot", "volumeSnapshot", snapshotMeta.Name, "namespace", snapshotMeta.Namespace)
	}

	src, err := sna.GetSource(ctx, name, namespace)
	if err != nil {
		return errkit.New("Failed to get source")
	}
	return sna.CreateFromSource(ctx, src, waitForReady, snapshotMeta, snapshotContentMeta)
}

// GetSource will return the CSI source that backs the volume snapshot.
func (sna *SnapshotBeta) GetSource(ctx context.Context, snapshotName, namespace string) (*Source, error) {
	return getSnapshotSource(ctx, sna.dynCli, v1beta1.VolSnapGVR, v1beta1.VolSnapContentGVR, snapshotName, namespace)
}

func getSnapshotSource(ctx context.Context, dynCli dynamic.Interface, snapGVR, snapContentGVR schema.GroupVersionResource, snapshotName, namespace string) (*Source, error) {
	snap, err := getSnapshot(ctx, dynCli, snapGVR, snapshotName, namespace)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to get snapshot", "volumeSnapshot", snapshotName)
	}
	if snap.Status.ReadyToUse == nil || !*snap.Status.ReadyToUse {
		return nil, errkit.New("Snapshot is not ready", "volumeSnapshot", snapshotName, "namespace", namespace)
	}
	if snap.Status.BoundVolumeSnapshotContentName == nil {
		return nil, errkit.New("Snapshot does not have content", "volumeSnapshot", snapshotName, "namespace", namespace)
	}

	cont, err := getSnapshotContent(ctx, dynCli, snapContentGVR, *snap.Status.BoundVolumeSnapshotContentName)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to get snapshot content", "volumeSnapshot", snapshotName, "volumeSnapshotContent", *snap.Status.BoundVolumeSnapshotContentName)
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
func (sna *SnapshotBeta) CreateFromSource(ctx context.Context, source *Source, waitForReady bool, snapshotMeta, snapshotContentMeta ObjectMeta) error {
	deletionPolicy, err := getDeletionPolicyFromClass(sna.dynCli, v1beta1.VolSnapClassGVR, source.VolumeSnapshotClassName)
	if err != nil {
		return errkit.Wrap(err, "Failed to get DeletionPolicy from VolumeSnapshotClass")
	}
	snapshotContentMeta.Name = snapshotMeta.Name + "-content-" + string(uuid.NewUUID())
	snapshotMeta.Labels = blockstorage.SanitizeTags(snapshotMeta.Labels)
	snap := UnstructuredVolumeSnapshot(v1beta1.VolSnapGVR, "", source.VolumeSnapshotClassName, snapshotMeta, snapshotContentMeta)
	if err := sna.CreateContentFromSource(ctx, source, snapshotMeta.Name, snapshotMeta.Namespace, deletionPolicy, snapshotContentMeta); err != nil {
		return err
	}
	if _, err := sna.dynCli.Resource(v1beta1.VolSnapGVR).Namespace(snapshotMeta.Namespace).Create(ctx, snap, metav1.CreateOptions{}); err != nil {
		return errkit.Wrap(err, "Failed to create content", "volumeSnapshot", snap.GetName())
	}
	if !waitForReady {
		return nil
	}
	err = sna.WaitOnReadyToUse(ctx, snapshotMeta.Name, snapshotMeta.Namespace)
	return err
}

// UpdateVolumeSnapshotStatusBeta sets the readyToUse valuse of a VolumeSnapshot.
func (sna *SnapshotBeta) UpdateVolumeSnapshotStatusBeta(ctx context.Context, namespace string, snapshotName string, readyToUse bool) error {
	return updateVolumeSnapshotStatus(ctx, sna.dynCli, v1beta1.VolSnapGVR, namespace, snapshotName, readyToUse)
}

func (sna *SnapshotBeta) GroupVersion(ctx context.Context) schema.GroupVersion {
	return schema.GroupVersion{
		Group:   v1beta1.GroupName,
		Version: v1beta1.Version,
	}
}

func updateVolumeSnapshotStatus(ctx context.Context, dynCli dynamic.Interface, snapGVR schema.GroupVersionResource, namespace string, snapshotName string, readyToUse bool) error {
	us, err := dynCli.Resource(snapGVR).Namespace(namespace).Get(ctx, snapshotName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	status, ok := us.Object["status"].(map[string]interface{})
	if !ok {
		return errkit.New("Failed to convert status to map", "volumeSnapshot", snapshotName, "status", status)
	}
	status["readyToUse"] = readyToUse
	us.Object["status"] = status
	if _, err := dynCli.Resource(snapGVR).Namespace(namespace).UpdateStatus(ctx, us, metav1.UpdateOptions{}); err != nil {
		return errkit.Wrap(err, "Failed to update status", "volumeSnapshot", snapshotName)
	}
	return nil
}

// CreateContentFromSource will create a 'VolumesnaphotContent' for the underlying snapshot source.
func (sna *SnapshotBeta) CreateContentFromSource(ctx context.Context, source *Source, snapshotName, namespace, deletionPolicy string, snapshotContentMeta ObjectMeta) error {
	content := UnstructuredVolumeSnapshotContent(v1beta1.VolSnapContentGVR, snapshotName, namespace, deletionPolicy, source.Driver, source.Handle, source.VolumeSnapshotClassName, snapshotContentMeta)
	if _, err := sna.dynCli.Resource(v1beta1.VolSnapContentGVR).Create(ctx, content, metav1.CreateOptions{}); err != nil {
		return errkit.Wrap(err, "Failed to create content", "volumeSnapshotContent", content.GetName())
	}
	return nil
}

// WaitOnReadyToUse will block until the Volumesnapshot in 'namespace' with name 'snapshotName'
// has status 'ReadyToUse' or 'ctx.Done()' is signalled.
func (sna *SnapshotBeta) WaitOnReadyToUse(ctx context.Context, snapshotName, namespace string) error {
	return waitOnReadyToUse(ctx, sna.dynCli, v1beta1.VolSnapGVR, snapshotName, namespace, isReadyToUseBeta)
}

func isReadyToUseBeta(us *unstructured.Unstructured) (bool, error) {
	vs := v1beta1.VolumeSnapshot{}
	if err := TransformUnstructured(us, &vs); err != nil {
		return false, err
	}
	if vs.Status == nil {
		return false, nil
	}
	// Error can be set while waiting for creation
	if vs.Status.Error != nil {
		return false, errkit.New(*vs.Status.Error.Message)
	}
	return (vs.Status.ReadyToUse != nil && *vs.Status.ReadyToUse && vs.Status.CreationTime != nil), nil
}

func getSnapshotContent(ctx context.Context, dynCli dynamic.Interface, snapContentGVR schema.GroupVersionResource, contentName string) (*v1.VolumeSnapshotContent, error) {
	us, err := dynCli.Resource(snapContentGVR).Get(ctx, contentName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	vsc := v1.VolumeSnapshotContent{}
	if err := TransformUnstructured(us, &vsc); err != nil {
		return nil, err
	}
	return &vsc, nil
}

func getDeletionPolicyFromClass(dynCli dynamic.Interface, snapClassGVR schema.GroupVersionResource, snapClassName string) (string, error) {
	us, err := dynCli.Resource(snapClassGVR).Get(context.TODO(), snapClassName, metav1.GetOptions{})
	if err != nil {
		return "", errkit.Wrap(err, "Failed to find VolumeSnapshotClass", "volumeSnapshotClass", snapClassName)
	}
	vsc := v1beta1.VolumeSnapshotClass{}
	if err := TransformUnstructured(us, &vsc); err != nil {
		return "", err
	}
	return vsc.DeletionPolicy, nil
}

// UnstructuredVolumeSnapshot returns Unstructured object for the VolumeSnapshot resource.
// If snapshotContentMeta has name value set, UnstructuredVolumeSnapshot will return VolumeSnapshot object with VolumeSnapshotContent information.
func UnstructuredVolumeSnapshot(gvr schema.GroupVersionResource, pvcName, snapClassName string, snapshotMeta, snapshotContentMeta ObjectMeta) *unstructured.Unstructured {
	snap := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": fmt.Sprintf("%s/%s", gvr.Group, gvr.Version),
			"kind":       VolSnapKind,
			"metadata": map[string]interface{}{
				"name":      snapshotMeta.Name,
				"namespace": snapshotMeta.Namespace,
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
	if snapshotContentMeta.Name != "" {
		snap.Object["spec"] = map[string]interface{}{
			"source": map[string]interface{}{
				"volumeSnapshotContentName": snapshotContentMeta.Name,
			},
			"volumeSnapshotClassName": snapClassName,
		}
	}
	if snapshotMeta.Labels != nil {
		snap.SetLabels(snapshotMeta.Labels)
	}
	if snapshotMeta.Annotations != nil {
		snap.SetAnnotations(snapshotMeta.Annotations)
	}
	return snap
}

// UnstructuredVolumeSnapshotContent returns Unstructured object for the VolumeSnapshotContent resource.
func UnstructuredVolumeSnapshotContent(gvr schema.GroupVersionResource, snapshotName, snapshotNS, deletionPolicy, driver, handle, snapClassName string, snapshotContentMeta ObjectMeta) *unstructured.Unstructured {
	snapshotContent := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": fmt.Sprintf("%s/%s", gvr.Group, gvr.Version),
			"kind":       VolSnapContentKind,
			"metadata": map[string]interface{}{
				"name": snapshotContentMeta.Name,
			},
			"spec": map[string]interface{}{
				"volumeSnapshotRef": map[string]interface{}{
					"kind":      VolSnapKind,
					"name":      snapshotName,
					"namespace": snapshotNS,
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
	if snapshotContentMeta.Labels != nil {
		snapshotContent.SetLabels(snapshotContentMeta.Labels)
	}
	if snapshotContentMeta.Annotations != nil {
		snapshotContent.SetAnnotations(snapshotContentMeta.Annotations)
	}
	return &snapshotContent
}

func UnstructuredVolumeSnapshotClass(gvr schema.GroupVersionResource, name, driver, deletionPolicy string, params map[string]string) *unstructured.Unstructured {
	obj := map[string]interface{}{
		"apiVersion": fmt.Sprintf("%s/%s", gvr.Group, gvr.Version),
		"kind":       VolSnapClassKind,
		"metadata": map[string]interface{}{
			"name": name,
		},
		VolSnapClassBetaDriverKey: driver,
		"deletionPolicy":          deletionPolicy,
	}
	if params != nil {
		obj["parameters"] = Mss2msi(params)
	}

	return &unstructured.Unstructured{
		Object: obj,
	}
}

// Mss2msi takes a map of string:string and returns a string:inteface map.
// This is useful since the unstructured type take map[string]interface{} as values.
func Mss2msi(in map[string]string) map[string]interface{} {
	if in == nil {
		return nil
	}
	paramsMap := map[string]interface{}{}
	for k, v := range in {
		paramsMap[k] = v
	}
	return paramsMap
}
