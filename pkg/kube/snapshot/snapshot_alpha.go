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
	"github.com/kanisterio/kanister/pkg/kube/snapshot/apis/v1alpha1"
)

const (
	PVCKind = "PersistentVolumeClaim"

	// Snapshot resource Kinds
	VolSnapClassKind                  = "VolumeSnapshotClass"
	VolSnapKind                       = "VolumeSnapshot"
	VolSnapContentKind                = "VolumeSnapshotContent"
	VolSnapClassAlphaDriverKey        = "snapshotter"
	VolSnapClassBetaDriverKey         = "driver"
	DeletionPolicyDelete              = "Delete"
	DeletionPolicyRetain              = "Retain"
	CloneVolumeSnapshotClassLabelName = "kanister-cloned-from"
)

type SnapshotAlpha struct {
	kubeCli kubernetes.Interface
	dynCli  dynamic.Interface
}

func NewSnapshotAlpha(kubeCli kubernetes.Interface, dynCli dynamic.Interface) Snapshotter {
	return &SnapshotAlpha{kubeCli: kubeCli, dynCli: dynCli}
}

// CloneVolumeSnapshotClass creates a copy of the source volume snapshot class
func (sna *SnapshotAlpha) CloneVolumeSnapshotClass(ctx context.Context, sourceClassName, targetClassName, newDeletionPolicy string, excludeAnnotations []string) error {
	usSourceSnapClass, err := sna.dynCli.Resource(v1alpha1.VolSnapClassGVR).Get(ctx, sourceClassName, metav1.GetOptions{})
	if err != nil {
		return errkit.Wrap(err, "Failed to find source VolumeSnapshotClass", "className", sourceClassName)
	}
	sourceSnapClass := v1alpha1.VolumeSnapshotClass{}
	if err := TransformUnstructured(usSourceSnapClass, &sourceSnapClass); err != nil {
		return err
	}
	existingAnnotations := sourceSnapClass.GetAnnotations()
	for _, key := range excludeAnnotations {
		delete(existingAnnotations, key)
	}
	usNew := UnstructuredVolumeSnapshotClassAlpha(targetClassName, sourceSnapClass.Snapshotter, newDeletionPolicy, sourceSnapClass.Parameters)
	// Set Annotations/Labels
	usNew.SetAnnotations(existingAnnotations)
	usNew.SetLabels(map[string]string{CloneVolumeSnapshotClassLabelName: sourceClassName})
	if _, err = sna.dynCli.Resource(v1alpha1.VolSnapClassGVR).Create(ctx, usNew, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		return errkit.Wrap(err, "Failed to create VolumeSnapshotClass", "className", targetClassName)
	}
	return nil
}

// GetVolumeSnapshotClass returns VolumeSnapshotClass name which is annotated with given key.
func (sna *SnapshotAlpha) GetVolumeSnapshotClass(ctx context.Context, annotationKey, annotationValue, storageClassName string) (string, error) {
	return GetSnapshotClassbyAnnotation(ctx, sna.dynCli, sna.kubeCli, v1alpha1.VolSnapClassGVR, annotationKey, annotationValue, storageClassName)
}

// Create creates a VolumeSnapshot and returns it or any error that happened meanwhile.
func (sna *SnapshotAlpha) Create(ctx context.Context, pvcName string, snapshotClass *string, waitForReady bool, snapshotMeta ObjectMeta) error {
	if _, err := sna.kubeCli.CoreV1().PersistentVolumeClaims(snapshotMeta.Namespace).Get(ctx, pvcName, metav1.GetOptions{}); err != nil {
		if apierrors.IsNotFound(err) {
			return errkit.New("Failed to find PVC", "pvc", pvcName, "namespace", snapshotMeta.Namespace)
		}
		return errkit.Wrap(err, "Failed to query PVC", "pvc", pvcName, "namespace", snapshotMeta.Namespace)
	}
	snapshotMeta.Labels = blockstorage.SanitizeTags(snapshotMeta.Labels)
	snap := UnstructuredVolumeSnapshotAlpha(pvcName, *snapshotClass, snapshotMeta, ObjectMeta{})
	if _, err := sna.dynCli.Resource(v1alpha1.VolSnapGVR).Namespace(snapshotMeta.Namespace).Create(ctx, snap, metav1.CreateOptions{}); err != nil {
		return err
	}

	if !waitForReady {
		return nil
	}

	if err := sna.WaitOnReadyToUse(ctx, snapshotMeta.Name, snapshotMeta.Namespace); err != nil {
		return err
	}

	_, err := sna.Get(ctx, snapshotMeta.Name, snapshotMeta.Namespace)
	return err
}

// Get will return the VolumeSnapshot in the 'namespace' with given 'name'.
func (sna *SnapshotAlpha) Get(ctx context.Context, name, namespace string) (*v1.VolumeSnapshot, error) {
	us, err := sna.dynCli.Resource(v1alpha1.VolSnapGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return TransformUnstructuredSnaphotV1alphaToV1(us)
}

func (sna *SnapshotAlpha) List(ctx context.Context, namespace string, labels map[string]string) (*v1.VolumeSnapshotList, error) {
	listOptions := metav1.ListOptions{}
	if labels != nil {
		labelSelector := metav1.LabelSelector{MatchLabels: blockstorage.SanitizeTags(labels)}
		listOptions.LabelSelector = pkglabels.Set(labelSelector.MatchLabels).String()
	}
	usList, err := sna.dynCli.Resource(v1alpha1.VolSnapGVR).Namespace(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, err
	}
	vsList := &v1.VolumeSnapshotList{}
	for _, us := range usList.Items {
		vs, err := TransformUnstructuredSnaphotV1alphaToV1(&us)
		if err != nil {
			return nil, err
		}
		vsList.Items = append(vsList.Items, *vs)
	}
	return vsList, nil
}

func TransformUnstructuredSnaphotV1alphaToV1(u *unstructured.Unstructured) (*v1.VolumeSnapshot, error) {
	vs := &v1alpha1.VolumeSnapshot{}
	if err := TransformUnstructured(u, vs); err != nil {
		return nil, err
	}

	// Populate v1.VolumeSnapshot object from v1alpha1.VolumeSnapshot
	vsRet := v1.VolumeSnapshot{}
	meta := vs.ObjectMeta.DeepCopy()
	if meta == nil {
		return nil, errkit.New("Invalid VolumeSnapshotObject: ObjectMeta is nil")
	}
	vsRet.ObjectMeta = *meta

	if vs.Spec.Source != nil && vs.Spec.Source.Kind == "PersistentVolumeClaim" {
		vsRet.Spec.Source.PersistentVolumeClaimName = &vs.Spec.Source.Name
	}
	if vs.Spec.VolumeSnapshotClassName != "" {
		vsRet.Spec.VolumeSnapshotClassName = &vs.Spec.VolumeSnapshotClassName
	}
	if vs.Spec.SnapshotContentName != "" {
		vsRet.Spec.Source.VolumeSnapshotContentName = &vs.Spec.SnapshotContentName
	}

	if vs.Status == (v1alpha1.VolumeSnapshotStatus{}) {
		return &vsRet, nil
	}

	// If Status is not nil, set VolumeSnapshotContentName from status
	vsRet.Status = &v1.VolumeSnapshotStatus{
		CreationTime: vs.Status.CreationTime,
		RestoreSize:  vs.Status.RestoreSize,
	}
	if vs.Spec.SnapshotContentName != "" {
		vsRet.Status.BoundVolumeSnapshotContentName = &vs.Spec.SnapshotContentName
	}
	if vs.Status.ReadyToUse {
		vsRet.Status.ReadyToUse = &vs.Status.ReadyToUse
	}
	if vs.Status.Error != nil {
		vsRet.Status.Error = &v1.VolumeSnapshotError{
			Time:    &vs.Status.Error.Time,
			Message: &vs.Status.Error.Message,
		}
	}
	return &vsRet, nil
}

// Delete will delete the VolumeSnapshot and returns any error as a result.
func (sna *SnapshotAlpha) Delete(ctx context.Context, name, namespace string) (*v1.VolumeSnapshot, error) {
	snap, err := sna.Get(ctx, name, namespace)
	if apierrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to find VolumeSnapshot", "namespace", namespace, "name", name)
	}
	if err := sna.dynCli.Resource(v1alpha1.VolSnapGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return nil, errkit.Wrap(err, "Failed to delete VolumeSnapshot", "namespace", namespace, "name", name)
	}
	// If the Snapshot does not exist, that's an acceptable error and we ignore it
	return snap, nil
}

// DeleteContent will delete the specified VolumeSnapshotContent
func (sna *SnapshotAlpha) DeleteContent(ctx context.Context, name string) error {
	if err := sna.dynCli.Resource(v1alpha1.VolSnapContentGVR).Delete(ctx, name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return errkit.Wrap(err, "Failed to delete VolumeSnapshotContent", "name", name)
	}
	// If the Snapshot Content does not exist, that's an acceptable error and we ignore it
	return nil
}

// Clone will clone the VolumeSnapshot to namespace 'cloneNamespace'.
// Underlying VolumeSnapshotContent will be cloned with a different name.
func (sna *SnapshotAlpha) Clone(ctx context.Context, name, namespace string, waitForReady bool, snapshotMeta, snapshotContentMeta ObjectMeta) error {
	_, err := sna.Get(ctx, snapshotMeta.Name, snapshotMeta.Namespace)
	if err == nil {
		return errkit.New("Target snapshot already exists in target namespace", "name", snapshotMeta.Name, "namespace", snapshotMeta.Namespace)
	}
	if !apierrors.IsNotFound(err) {
		return errkit.Wrap(err, "Failed to query target Volumesnapshot", "name", snapshotMeta.Name, "namespace", snapshotMeta.Namespace)
	}

	src, err := sna.GetSource(ctx, name, namespace)
	if err != nil {
		return errkit.Wrap(err, "Failed to get source")
	}
	return sna.CreateFromSource(ctx, src, waitForReady, snapshotMeta, snapshotContentMeta)
}

// GetSource will return the CSI source that backs the volume snapshot.
func (sna *SnapshotAlpha) GetSource(ctx context.Context, snapshotName, namespace string) (*Source, error) {
	snap, err := sna.Get(ctx, snapshotName, namespace)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to get snapshot", "name", snapshotName, "namespace", namespace)
	}
	if snap.Status.ReadyToUse == nil || !*snap.Status.ReadyToUse {
		return nil, errkit.New("Snapshot is not ready", "name", snapshotName, "namespace", namespace)
	}
	if snap.Status.BoundVolumeSnapshotContentName == nil {
		return nil, errkit.New("Snapshot does not have content", "name", snapshotName, "namespace", namespace)
	}

	cont, err := sna.getContent(ctx, *snap.Status.BoundVolumeSnapshotContentName)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to get snapshot content", "name", snapshotName, "contentName", *snap.Status.BoundVolumeSnapshotContentName)
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
func (sna *SnapshotAlpha) CreateFromSource(ctx context.Context, source *Source, waitForReady bool, snapshotMeta, snapshotContentMeta ObjectMeta) error {
	deletionPolicy, err := sna.getDeletionPolicyFromClass(source.VolumeSnapshotClassName)
	if err != nil {
		return errkit.Wrap(err, "Failed to get DeletionPolicy from VolumeSnapshotClass")
	}
	snapshotContentMeta.Name = snapshotMeta.Name + "-content-" + string(uuid.NewUUID())
	snapshotMeta.Labels = blockstorage.SanitizeTags(snapshotMeta.Labels)
	snap := UnstructuredVolumeSnapshotAlpha("", source.VolumeSnapshotClassName, snapshotMeta, snapshotContentMeta)
	if err := sna.CreateContentFromSource(ctx, source, snapshotMeta.Name, snapshotMeta.Namespace, deletionPolicy, snapshotContentMeta); err != nil {
		return err
	}
	if _, err := sna.dynCli.Resource(v1alpha1.VolSnapGVR).Namespace(snapshotMeta.Namespace).Create(ctx, snap, metav1.CreateOptions{}); err != nil {
		return errkit.Wrap(err, "Failed to create content", "name", snap.GetName(), "namespace", snapshotMeta.Namespace)
	}
	if !waitForReady {
		return nil
	}
	return sna.WaitOnReadyToUse(ctx, snapshotMeta.Name, snapshotMeta.Namespace)
}

// UpdateVolumeSnapshotStatusAlpha sets the readyToUse valuse of a VolumeSnapshot.
func (sna *SnapshotAlpha) UpdateVolumeSnapshotStatusAlpha(ctx context.Context, namespace string, snapshotName string, readyToUse bool) error {
	us, err := sna.dynCli.Resource(v1alpha1.VolSnapGVR).Namespace(namespace).Get(ctx, snapshotName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	status, ok := us.Object["status"].(map[string]interface{})
	if !ok {
		return errkit.New("Failed to convert status to map", "name", snapshotName, "status", status)
	}
	status["readyToUse"] = readyToUse
	us.Object["status"] = status
	if _, err := sna.dynCli.Resource(v1alpha1.VolSnapGVR).Namespace(namespace).UpdateStatus(ctx, us, metav1.UpdateOptions{}); err != nil {
		return errkit.Wrap(err, "Failed to update status", "name", snapshotName)
	}
	return nil
}

// CreateContentFromSource will create a 'VolumesnaphotContent' for the underlying snapshot source.
func (sna *SnapshotAlpha) CreateContentFromSource(ctx context.Context, source *Source, snapshotName, namespace, deletionPolicy string, snapshotContentMeta ObjectMeta) error {
	content := UnstructuredVolumeSnapshotContentAlpha(snapshotName, namespace, deletionPolicy, source.Driver, source.Handle, source.VolumeSnapshotClassName, snapshotContentMeta)
	if _, err := sna.dynCli.Resource(v1alpha1.VolSnapContentGVR).Create(ctx, content, metav1.CreateOptions{}); err != nil {
		return errkit.Wrap(err, "Failed to create content", "contentName", content.GetName())
	}
	return nil
}

func isReadyToUseAlpha(us *unstructured.Unstructured) (bool, error) {
	vs := v1alpha1.VolumeSnapshot{}
	if err := TransformUnstructured(us, &vs); err != nil {
		return false, err
	}
	// Error can be set while waiting for creation
	if vs.Status.Error != nil {
		return false, errkit.New(vs.Status.Error.Message)
	}
	return (vs.Status.ReadyToUse && vs.Status.CreationTime != nil), nil
}

// WaitOnReadyToUse will block until the Volumesnapshot in namespace 'namespace' with name 'snapshotName'
// has status 'ReadyToUse' or 'ctx.Done()' is signalled.
func (sna *SnapshotAlpha) WaitOnReadyToUse(ctx context.Context, snapshotName, namespace string) error {
	return waitOnReadyToUse(ctx, sna.dynCli, v1alpha1.VolSnapGVR, snapshotName, namespace, isReadyToUseAlpha)
}

func (sna *SnapshotAlpha) GroupVersion(ctx context.Context) schema.GroupVersion {
	return schema.GroupVersion{
		Group:   v1alpha1.GroupName,
		Version: v1alpha1.Version,
	}
}
func (sna *SnapshotAlpha) getContent(ctx context.Context, contentName string) (*v1alpha1.VolumeSnapshotContent, error) {
	us, err := sna.dynCli.Resource(v1alpha1.VolSnapContentGVR).Get(ctx, contentName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	vsc := v1alpha1.VolumeSnapshotContent{}
	if err := TransformUnstructured(us, &vsc); err != nil {
		return nil, err
	}
	return &vsc, nil
}

func (sna *SnapshotAlpha) getDeletionPolicyFromClass(snapClassName string) (string, error) {
	us, err := sna.dynCli.Resource(v1alpha1.VolSnapClassGVR).Get(context.TODO(), snapClassName, metav1.GetOptions{})
	if err != nil {
		return "", errkit.Wrap(err, "Failed to find VolumeSnapshotClass", "className", snapClassName)
	}
	vsc := v1alpha1.VolumeSnapshotClass{}
	if err := TransformUnstructured(us, &vsc); err != nil {
		return "", err
	}
	return vsc.DeletionPolicy, nil
}

// UnstructuredVolumeSnapshotAlpha returns Unstructured object for the VolumeSnapshot resource.
// If snapshotContentMeta has name value set, UnstructuredVolumeSnapshotAlpha will return VolumeSnapshot object with VolumeSnapshotContent information.
func UnstructuredVolumeSnapshotAlpha(pvcName, snapClassName string, snapshotMeta, snapshotContentMeta ObjectMeta) *unstructured.Unstructured {
	snap := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": fmt.Sprintf("%s/%s", v1alpha1.GroupName, v1alpha1.Version),
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
				"kind":      PVCKind,
				"name":      pvcName,
				"namespace": snapshotMeta.Namespace,
			},
			"snapshotClassName": snapClassName,
			"deletionPolicy":    "Delete",
		}
	}
	if snapshotContentMeta.Name != "" {
		snap.Object["spec"] = map[string]interface{}{
			"snapshotContentName": snapshotContentMeta.Name,
			"snapshotClassName":   snapClassName,
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

// UnstructuredVolumeSnapshotContentAlpha returns Unstructured object for the VolumeSnapshotContent resource.
func UnstructuredVolumeSnapshotContentAlpha(snapshotName, snapshotNS, deletionPolicy, driver, handle, snapClassName string, snapshotContentMeta ObjectMeta) *unstructured.Unstructured {
	snaphotContent := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": fmt.Sprintf("%s/%s", v1alpha1.GroupName, v1alpha1.Version),
			"kind":       VolSnapContentKind,
			"metadata": map[string]interface{}{
				"name": snapshotContentMeta.Name,
			},
			"spec": map[string]interface{}{
				"csiVolumeSnapshotSource": map[string]interface{}{
					"driver":         driver,
					"snapshotHandle": handle,
				},
				"volumeSnapshotRef": map[string]interface{}{
					"kind":      VolSnapKind,
					"name":      snapshotName,
					"namespace": snapshotNS,
				},
				"snapshotClassName": snapClassName,
				"deletionPolicy":    deletionPolicy,
			},
		},
	}
	if snapshotContentMeta.Labels != nil {
		snaphotContent.SetLabels(snapshotContentMeta.Labels)
	}
	if snapshotContentMeta.Annotations != nil {
		snaphotContent.SetAnnotations(snapshotContentMeta.Annotations)
	}
	return &snaphotContent
}

func UnstructuredVolumeSnapshotClassAlpha(name, driver, deletionPolicy string, params map[string]string) *unstructured.Unstructured {
	obj := map[string]interface{}{
		"apiVersion": fmt.Sprintf("%s/%s", v1alpha1.GroupName, v1alpha1.Version),
		"kind":       VolSnapClassKind,
		"metadata": map[string]interface{}{
			"name": name,
		},
		VolSnapClassAlphaDriverKey: driver,
		"deletionPolicy":           deletionPolicy,
	}

	if params != nil {
		obj["parameters"] = Mss2msi(params)
	}

	return &unstructured.Unstructured{
		Object: obj,
	}
}

// TransformUnstructured maps Unstructured object to object pointed by obj
func TransformUnstructured(u *unstructured.Unstructured, obj metav1.Object) error {
	if u == nil {
		return errkit.New("Cannot deserialize nil unstructured")
	}
	b, err := json.Marshal(u.Object)
	if err != nil {
		gvk := u.GetObjectKind().GroupVersionKind()
		return errkit.Wrap(err, "Failed to Marshal unstructured object GroupVersionKind", "unstructured", gvk)
	}
	err = json.Unmarshal(b, obj)
	if err != nil {
		return errkit.Wrap(err, "Failed to Unmarshal unstructured object")
	}

	return nil
}

// GetSnapshotClassbyAnnotation checks if the provided annotation is present in either the storageclass
// or volumesnapshotclass and returns the volumesnapshotclass.
func GetSnapshotClassbyAnnotation(ctx context.Context, dynCli dynamic.Interface, kubeCli kubernetes.Interface, gvr schema.GroupVersionResource, annotationKey, annotationValue, storageClass string) (string, error) {
	// fetch storageClass
	sc, err := kubeCli.StorageV1().StorageClasses().Get(ctx, storageClass, metav1.GetOptions{})
	if err != nil {
		return "", errkit.Wrap(err, "Failed to find StorageClass in the cluster", "class", storageClass)
	}
	// Check if storageclass annotation override is present.
	if val, ok := sc.Annotations[annotationKey]; ok {
		vsc, err := dynCli.Resource(gvr).Get(ctx, val, metav1.GetOptions{})
		if err != nil {
			return "", errkit.Wrap(err, "Failed to get VolumeSnapshotClass specified in Storageclass annotations", "snapshotClass", val, "storageClass", sc.Name)
		}
		return vsc.GetName(), nil
	}
	us, err := dynCli.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", errkit.Wrap(err, "Failed to get VolumeSnapshotClasses in the cluster")
	}
	if us == nil || len(us.Items) == 0 {
		return "", errkit.New("Failed to find any VolumeSnapshotClass in the cluster")
	}
	for _, vsc := range us.Items {
		ans := vsc.GetAnnotations()
		driver, err := getDriverFromUnstruturedVSC(vsc)
		if err != nil {
			return "", errkit.Wrap(err, "Failed to get driver for VolumeSnapshotClass", "className", vsc.GetName())
		}
		if val, ok := ans[annotationKey]; ok && val == annotationValue && driver == sc.Provisioner {
			return vsc.GetName(), nil
		}
	}
	return "", errkit.New("Failed to find VolumeSnapshotClass with annotation in the cluster", "annotationKey", annotationKey, "annotationValue", annotationValue)
}

func getDriverFromUnstruturedVSC(uVSC unstructured.Unstructured) (string, error) {
	if uVSC.GetKind() != VolSnapClassKind {
		return "", errkit.New("Cannot get diver for kind", "kind", uVSC.GetKind())
	}
	driver, ok := uVSC.Object[VolSnapClassAlphaDriverKey]
	if !ok {
		driver, ok = uVSC.Object[VolSnapClassBetaDriverKey]
	}
	if !ok {
		return "", errkit.New("VolumeSnapshotClass missing driver/snapshotter field", "volumeSnapshotClass", uVSC.GetName())
	}
	if driverString, ok := driver.(string); ok {
		return driverString, nil
	}
	return "", errkit.New("Failed to convert driver to string")
}
