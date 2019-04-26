package volume

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	snapshotclient "github.com/kubernetes-csi/external-snapshotter/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/blockstorage"
	"github.com/kanisterio/kanister/pkg/poll"
)

const (
	pvMatchLabelName = "kanisterpvmatchid"
	pvcGenerateName  = "kanister-pvc-"
	// PVZoneLabelName is a known k8s label. used to specify volume zone
	PVZoneLabelName = "failure-domain.beta.kubernetes.io/zone"
	// PVRegionLabelName is a known k8s label
	PVRegionLabelName = "failure-domain.beta.kubernetes.io/region"
	// NoPVCNameSpecified is used by the caller to indicate that the PVC name
	// should be auto-generated
	NoPVCNameSpecified = ""
)

// CreatePVC creates a PersistentVolumeClaim and returns its name
// An empty 'targetVolID' indicates the caller would like the PV to be dynamically provisioned
// An empty 'name' indicates the caller would like the name to be auto-generated
// An error indicating that the PVC already exists is ignored (for idempotency)
func CreatePVC(ctx context.Context, kubeCli kubernetes.Interface, ns string, name string, sizeGB int64, targetVolID string, annotations map[string]string) (string, error) {
	sizeFmt := fmt.Sprintf("%dGi", sizeGB)
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
	createdPVC, err := kubeCli.CoreV1().PersistentVolumeClaims(ns).Create(&pvc)
	if err != nil {
		if name != "" && apierrors.IsAlreadyExists(err) {
			return name, nil
		}
		return "", errors.Wrapf(err, "Unable to create PVC %v", pvc)
	}
	return createdPVC.Name, nil
}

// CreatePVCFromSnapshot will restore a volume and returns the resulting
// PersistentVolumeClaim and any error that happened in the process.
//
// 'volumeName' is the name of the PVC that will be restored from the snapshot.
// 'snapshotName' is the name of the VolumeSnapshot that will be used for restoring.
// 'namespace' is the namespace of the VolumeSnapshot. The PVC will be restored to the same namepsace.
// 'restoreSize' will override existing restore size from snapshot content if provided.
func CreatePVCFromSnapshot(ctx context.Context, kubeCli kubernetes.Interface, snapCli snapshotclient.Interface, namespace, volumeName, snapshotName string, restoreSize *int) (string, error) {
	snap, err := snapCli.VolumesnapshotV1alpha1().VolumeSnapshots(namespace).Get(snapshotName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	size := snap.Status.RestoreSize
	if restoreSize != nil {
		s := resource.MustParse(fmt.Sprintf("%dGi", *restoreSize))
		size = &s
	}
	if size == nil {
		return "", fmt.Errorf("Restore size is empty and no restore size argument given, Volumesnapshot: %s", snap.Name)
	}

	snapshotKind := "VolumeSnapshot"
	snapshotAPIGroup := "snapshot.storage.k8s.io"
	pvc := &v1.PersistentVolumeClaim{
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			DataSource: &v1.TypedLocalObjectReference{
				APIGroup: &snapshotAPIGroup,
				Kind:     snapshotKind,
				Name:     snapshotName,
			},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: *size,
				},
			},
		},
	}
	if volumeName != "" {
		pvc.ObjectMeta.Name = volumeName
	} else {
		pvc.ObjectMeta.GenerateName = pvcGenerateName
	}

	pvc, err = kubeCli.CoreV1().PersistentVolumeClaims(namespace).Create(pvc)
	if err != nil {
		if volumeName != "" && apierrors.IsAlreadyExists(err) {
			return volumeName, nil
		}
		return "", errors.Wrapf(err, "Unable to create PVC, PVC: %v", pvc)
	}
	return pvc.Name, err
}

// CreatePV creates a PersistentVolume and returns its name
// For retry idempotency, checks whether PV associated with volume already exists
func CreatePV(ctx context.Context, kubeCli kubernetes.Interface, vol *blockstorage.Volume, volType blockstorage.Type, annotations map[string]string) (string, error) {
	sizeFmt := fmt.Sprintf("%dGi", vol.Size)
	size, err := resource.ParseQuantity(sizeFmt)
	if err != nil {
		return "", errors.Wrapf(err, "Unable to parse sizeFmt %s", sizeFmt)
	}
	matchLabels := map[string]string{pvMatchLabelName: filepath.Base(vol.ID)}

	// Since behavior and error returned from repeated create might vary, check first
	sel := labelSelector(matchLabels)
	options := metav1.ListOptions{LabelSelector: sel}
	pvl, err := kubeCli.CoreV1().PersistentVolumes().List(options)
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
		pv.ObjectMeta.Labels[PVZoneLabelName] = vol.Az
		pv.ObjectMeta.Labels[PVRegionLabelName] = zoneToRegion(vol.Az)
	case blockstorage.TypeGPD:
		pv.Spec.PersistentVolumeSource.GCEPersistentDisk = &v1.GCEPersistentDiskVolumeSource{
			PDName: vol.ID,
		}
		pv.ObjectMeta.Labels[PVZoneLabelName] = vol.Az
		pv.ObjectMeta.Labels[PVRegionLabelName] = zoneToRegion(vol.Az)
	default:
		return "", errors.Errorf("Volume type %v(%T) not supported ", volType, volType)
	}

	createdPV, err := kubeCli.CoreV1().PersistentVolumes().Create(&pv)
	if err != nil {
		return "", errors.Wrapf(err, "Unable to create PV for volume %v", pv)
	}
	return createdPV.Name, nil
}

// DeletePVC deletes the given PVC immediately and waits with timeout until it is returned as deleted
func DeletePVC(cli kubernetes.Interface, namespace, pvcName string) error {
	var now int64
	if err := cli.Core().PersistentVolumeClaims(namespace).Delete(pvcName, &metav1.DeleteOptions{GracePeriodSeconds: &now}); err != nil {
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
		_, err := cli.Core().PersistentVolumeClaims(namespace).Get(pvcName, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
}

var labelBlackList = map[string]struct{}{
	"chart":    struct{}{},
	"heritage": struct{}{},
}

func labelSelector(labels map[string]string) string {
	ls := make([]string, 0, len(labels))
	for k, v := range labels {
		if _, ok := labelBlackList[k]; ok {
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
