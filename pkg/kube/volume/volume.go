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

package volume

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/blockstorage"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/kube/snapshot"
	"github.com/kanisterio/kanister/pkg/poll"
)

const (
	pvMatchLabelName = "kanisterpvmatchid"
	pvcGenerateName  = "kanister-pvc-"
	// NoPVCNameSpecified is used by the caller to indicate that the PVC name
	// should be auto-generated
	NoPVCNameSpecified = ""
)

// CreatePVC creates a PersistentVolumeClaim and returns its name
// An empty 'targetVolID' indicates the caller would like the PV to be dynamically provisioned
// An empty 'name' indicates the caller would like the name to be auto-generated
// An error indicating that the PVC already exists is ignored (for idempotency)
func CreatePVC(ctx context.Context, kubeCli kubernetes.Interface, ns string, name string, sizeInBytes int64, targetVolID string, annotations map[string]string) (string, error) {
	sizeFmt := fmt.Sprintf("%d", sizeInBytes)
	size, err := resource.ParseQuantity(sizeFmt)
	emptyStorageClass := ""
	if err != nil {
		return "", errors.Wrapf(err, "Unable to parse sizeFmt %s", sizeFmt)
	}
	pvc := v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: annotations,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceName(v1.ResourceStorage): size,
				},
			},
		},
	}
	if name != "" {
		pvc.ObjectMeta.Name = name
	} else {
		pvc.ObjectMeta.GenerateName = pvcGenerateName
	}

	if targetVolID != "" {
		// If targetVolID is set, static provisioning is desired
		pvc.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: map[string]string{pvMatchLabelName: filepath.Base(targetVolID)},
		}
		// Disable dynamic provisioning by setting an empty storage
		pvc.Spec.StorageClassName = &emptyStorageClass
	}
	createdPVC, err := kubeCli.CoreV1().PersistentVolumeClaims(ns).Create(ctx, &pvc, metav1.CreateOptions{})
	if err != nil {
		if name != "" && apierrors.IsAlreadyExists(err) {
			return name, nil
		}
		return "", errors.Wrapf(err, "Unable to create PVC %v", pvc)
	}
	return createdPVC.Name, nil
}

// CreatePVCFromSnapshotArgs describes the arguments for CreatePVCFromSnapshot
// 'VolumeName' is the name of the PVC that will be restored from the snapshot.
// 'StorageClassName' is the name of the storage class used to create the PVC.
// 'SnapshotName' is the name of the VolumeSnapshot that will be used for restoring.
// 'Namespace' is the namespace of the VolumeSnapshot. The PVC will be restored to the same namepsace.
// 'RestoreSize' will override existing restore size from snapshot content if provided.
// 'Labels' will be added to the PVC.
type CreatePVCFromSnapshotArgs struct {
	KubeCli          kubernetes.Interface
	DynCli           dynamic.Interface
	Namespace        string
	VolumeName       string
	StorageClassName string
	SnapshotName     string
	RestoreSize      string
	Labels           map[string]string
}

// CreatePVCFromSnapshot will restore a volume and returns the resulting
// PersistentVolumeClaim and any error that happened in the process.
func CreatePVCFromSnapshot(ctx context.Context, args *CreatePVCFromSnapshotArgs) (string, error) {
	var size *resource.Quantity
	if args.RestoreSize == "" {
		sns, err := snapshot.NewSnapshotter(args.KubeCli, args.DynCli)
		if err != nil {
			return "", err
		}
		snap, err := sns.Get(ctx, args.SnapshotName, args.Namespace)
		if err != nil {
			return "", err
		}

		size = snap.Status.RestoreSize
	} else {
		s := resource.MustParse(args.RestoreSize)
		size = &s
	}

	if size == nil {
		return "", fmt.Errorf("Restore size is empty and no restore size argument given, Volumesnapshot: %s", args.SnapshotName)
	}

	snapshotKind := "VolumeSnapshot"
	snapshotAPIGroup := "snapshot.storage.k8s.io"
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Labels: args.Labels,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			DataSource: &v1.TypedLocalObjectReference{
				APIGroup: &snapshotAPIGroup,
				Kind:     snapshotKind,
				Name:     args.SnapshotName,
			},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: *size,
				},
			},
		},
	}
	if args.VolumeName != "" {
		pvc.ObjectMeta.Name = args.VolumeName
	} else {
		pvc.ObjectMeta.GenerateName = pvcGenerateName
	}
	if args.StorageClassName != "" {
		pvc.Spec.StorageClassName = &args.StorageClassName
	}

	pvc, err := args.KubeCli.CoreV1().PersistentVolumeClaims(args.Namespace).Create(ctx, pvc, metav1.CreateOptions{})
	if err != nil {
		if args.VolumeName != "" && apierrors.IsAlreadyExists(err) {
			return args.VolumeName, nil
		}
		return "", errors.Wrapf(err, "Unable to create PVC, PVC: %v", pvc)
	}
	return pvc.Name, err
}

// CreatePV creates a PersistentVolume and returns its name
// For retry idempotency, checks whether PV associated with volume already exists
func CreatePV(ctx context.Context, kubeCli kubernetes.Interface, vol *blockstorage.Volume, volType blockstorage.Type, annotations map[string]string) (string, error) {
	sizeFmt := fmt.Sprintf("%d", vol.SizeInBytes)
	size, err := resource.ParseQuantity(sizeFmt)
	if err != nil {
		return "", errors.Wrapf(err, "Unable to parse sizeFmt %s", sizeFmt)
	}
	matchLabels := map[string]string{pvMatchLabelName: filepath.Base(vol.ID)}

	// Since behavior and error returned from repeated create might vary, check first
	sel := labelSelector(matchLabels)
	options := metav1.ListOptions{LabelSelector: sel}
	pvl, err := kubeCli.CoreV1().PersistentVolumes().List(ctx, options)
	if err == nil && len(pvl.Items) == 1 {
		return pvl.Items[0].Name, nil
	}

	pv := v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kanister-pv-",
			Labels:       matchLabels,
			Annotations:  annotations,
		},
		Spec: v1.PersistentVolumeSpec{
			Capacity: v1.ResourceList{
				v1.ResourceName(v1.ResourceStorage): size,
			},
			AccessModes:                   []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			PersistentVolumeReclaimPolicy: v1.PersistentVolumeReclaimDelete,
		},
	}
	switch volType {
	case blockstorage.TypeEBS:
		pv.Spec.PersistentVolumeSource.AWSElasticBlockStore = &v1.AWSElasticBlockStoreVolumeSource{
			VolumeID: vol.ID,
		}
		pv.ObjectMeta.Labels[kube.FDZoneLabelName] = vol.Az
		pv.ObjectMeta.Labels[kube.FDRegionLabelName] = zoneToRegion(vol.Az)
	case blockstorage.TypeGPD:
		pv.Spec.PersistentVolumeSource.GCEPersistentDisk = &v1.GCEPersistentDiskVolumeSource{
			PDName: vol.ID,
		}
		pv.ObjectMeta.Labels[kube.FDZoneLabelName] = vol.Az
		pv.ObjectMeta.Labels[kube.FDRegionLabelName] = zoneToRegion(vol.Az)

	default:
		return "", errors.Errorf("Volume type %v(%T) not supported ", volType, volType)
	}

	createdPV, err := kubeCli.CoreV1().PersistentVolumes().Create(ctx, &pv, metav1.CreateOptions{})
	if err != nil {
		return "", errors.Wrapf(err, "Unable to create PV for volume %v", pv)
	}
	return createdPV.Name, nil
}

// DeletePVC deletes the given PVC immediately and waits with timeout until it is returned as deleted
func DeletePVC(cli kubernetes.Interface, namespace, pvcName string) error {
	var now int64
	if err := cli.CoreV1().PersistentVolumeClaims(namespace).Delete(context.TODO(), pvcName, metav1.DeleteOptions{GracePeriodSeconds: &now}); err != nil {
		// If the PVC does not exist, that's an acceptable error
		if !apierrors.IsNotFound(err) {
			return err
		}
	}

	// Check the pvc is not returned. If the expected condition is not met in time, PollImmediate will
	// return ErrWaitTimeout
	ctx, c := context.WithTimeout(context.TODO(), time.Minute)
	defer c()
	return poll.Wait(ctx, func(context.Context) (bool, error) {
		_, err := cli.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, pvcName, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
}

var labelDenyList = map[string]struct{}{
	"chart":    {},
	"heritage": {},
}

func labelSelector(labels map[string]string) string {
	ls := make([]string, 0, len(labels))
	for k, v := range labels {
		if _, ok := labelDenyList[k]; ok {
			continue
		}
		ls = append(ls, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(ls, ",")
}

// zoneToRegion removes -latter or just last latter from provided zone.
func zoneToRegion(zone string) string {
	r, _ := regexp.Compile("-?[a-z]$")
	return r.ReplaceAllString(zone, "")
}
