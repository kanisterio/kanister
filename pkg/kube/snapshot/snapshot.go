package snapshot

import (
	"context"

	snapshot "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1alpha1"
	snapshotclient "github.com/kubernetes-csi/external-snapshotter/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/poll"
)

const (
	snapshotKind = "VolumeSnapshot"
	pvcKind      = "PersistentVolumeClaim"
)

var snapshotAPIGroup = "snapshot.storage.k8s.io"

// Create creates a VolumeSnapshot and returns it or any error happened meanwhile.
//
// 'name' is the name of the VolumeSnapshot.
// 'namespace' is namespace of the PVC. VolumeSnapshot will be crated in the same namespace.
// 'volumeName' is the name of the PVC of which we will take snapshot. It must be in the same namespace 'ns'.
// 'waitForReady' will block the caller until the snapshot status is 'ReadyToUse'.
// or 'ctx.Done()' is signalled. Otherwise it will return immediately after the snapshot is cut.
func Create(ctx context.Context, kubeCli kubernetes.Interface, snapCli snapshotclient.Interface, name, namespace, volumeName string, snapshotClass *string, waitForReady bool) error {
	if _, err := kubeCli.CoreV1().PersistentVolumeClaims(namespace).Get(volumeName, metav1.GetOptions{}); err != nil {
		if k8errors.IsNotFound(err) {
			return errors.Errorf("Failed to find PVC %s, Namespace %s", volumeName, namespace)
		}
		return errors.Errorf("Failed to query PVC %s, Namespace %s: %v", volumeName, namespace, err)
	}

	snap := &snapshot.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: snapshot.VolumeSnapshotSpec{
			Source: &corev1.TypedLocalObjectReference{
				Kind: pvcKind,
				Name: volumeName,
			},
			VolumeSnapshotClassName: snapshotClass,
		},
	}

	snap, err := snapCli.VolumesnapshotV1alpha1().VolumeSnapshots(namespace).Create(snap)
	if err != nil {
		return err
	}

	if !waitForReady {
		return nil
	}

	err = WaitOnReadyToUse(ctx, snapCli, name, namespace)
	if err != nil {
		return err
	}

	_, err = Get(ctx, snapCli, name, namespace)
	return err
}

// Get will return the VolumeSnapshot in the namespace 'namespace' with given 'name'.
//
// 'name' is the name of the VolumeSnapshot that will be returned.
// 'namespace' is the namespace of the VolumeSnapshot that will be returned.
func Get(ctx context.Context, snapCli snapshotclient.Interface, name, namespace string) (*snapshot.VolumeSnapshot, error) {
	return snapCli.VolumesnapshotV1alpha1().VolumeSnapshots(namespace).Get(name, metav1.GetOptions{})
}

// Delete will delete the VolumeSnapshot and returns any error as a result.
//
// 'name' is the name of the VolumeSnapshot that will be deleted.
// 'namespace' is the namespace of the VolumeSnapshot that will be deleted.
func Delete(ctx context.Context, snapCli snapshotclient.Interface, name, namespace string) error {
	return snapCli.VolumesnapshotV1alpha1().VolumeSnapshots(namespace).Delete(name, &metav1.DeleteOptions{})
}

// Clone will clone the VolumeSnapshot to namespace 'cloneNamespace'.
// Underlying VolumeSnapshotContent will be cloned with a different name.
//
// 'name' is the name of the VolumeSnapshot that will be cloned.
// 'namespace' is the namespace of the VolumeSnapshot that will be cloned.
// 'cloneName' is name of the clone.
// 'cloneNamespace' is the namespace where the clone will be created.
// 'waitForReady' will make the function blocks until the clone's status is ready to use.
func Clone(ctx context.Context, snapCli snapshotclient.Interface, name, namespace, cloneName, cloneNamespace string, waitForReady bool) error {
	snap, err := Get(ctx, snapCli, name, namespace)
	if err != nil {
		return err
	}
	if !snap.Status.ReadyToUse {
		return errors.Errorf("Original snapshot is not ready, VolumeSnapshot: %s, Namespace: %s", cloneName, cloneNamespace)
	}
	if snap.Spec.SnapshotContentName == "" {
		return errors.Errorf("Original snapshot does not have content, VolumeSnapshot: %s, Namespace: %s", cloneName, cloneNamespace)
	}

	_, err = Get(ctx, snapCli, cloneName, cloneNamespace)
	if err == nil {
		return errors.Errorf("Target snapshot already exists in target namespace, Volumesnapshot: %s, Namespace: %s", cloneName, cloneNamespace)
	}
	if !k8errors.IsNotFound(err) {
		return errors.Errorf("Failed to query target Volumesnapshot: %s, Namespace: %s: %v", cloneName, cloneNamespace, err)
	}

	src, err := GetSource(ctx, snapCli, name, namespace)
	if err != nil {
		return errors.Errorf("Failed to get source")
	}
	return CreateFromSource(ctx, snapCli, src, cloneName, cloneNamespace, waitForReady)
}

// Source represents the CSI source of the Volumesnapshot.
type Source struct {
	Handle                  string
	Driver                  string
	RestoreSize             *int64
	VolumeSnapshotClassName *string
}

// GetSource will return the CSI source that backs the volume snapshot.
//
// 'snapshotName' is the name of the Volumesnapshot.
// 'namespace' is the namespace of the Volumesnapshot.
func GetSource(ctx context.Context, snapCli snapshotclient.Interface, snapshotName, namespace string) (*Source, error) {
	snap, err := Get(ctx, snapCli, snapshotName, namespace)
	if err != nil {
		return nil, errors.Errorf("Failed to get snapshot, VolumeSnapshot: %s, Error: %v", snapshotName, err)
	}
	cont, err := getContent(ctx, snapCli, snap.Spec.SnapshotContentName)
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
//
// 'source' contains information about CSI snapshot.
// 'snapshotName' is the name of the snapshot that will be created.
// 'namespace' is the namespace of the snapshot.
// 'waitForReady' blocks the caller until snapshot is ready to use or context is cancelled.
func CreateFromSource(ctx context.Context, snapCli snapshotclient.Interface, source *Source, snapshotName, namespace string, waitForReady bool) error {
	contentName := snapshotName + "-content-" + string(uuid.NewUUID())
	content := &snapshot.VolumeSnapshotContent{
		ObjectMeta: metav1.ObjectMeta{
			Name: contentName,
		},
		Spec: snapshot.VolumeSnapshotContentSpec{
			VolumeSnapshotSource: snapshot.VolumeSnapshotSource{
				CSI: &snapshot.CSIVolumeSnapshotSource{
					Driver:         source.Driver,
					SnapshotHandle: source.Handle,
				},
			},
			VolumeSnapshotRef: &corev1.ObjectReference{
				Kind:      snapshotKind,
				Namespace: namespace,
				Name:      snapshotName,
			},
			VolumeSnapshotClassName: source.VolumeSnapshotClassName,
		},
	}
	snap := &snapshot.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name: snapshotName,
		},
		Spec: snapshot.VolumeSnapshotSpec{
			SnapshotContentName:     content.Name,
			VolumeSnapshotClassName: content.Spec.VolumeSnapshotClassName,
		},
	}

	content, err := snapCli.VolumesnapshotV1alpha1().VolumeSnapshotContents().Create(content)
	if err != nil {
		return errors.Errorf("Failed to create content, VolumesnapshotContent: %s, Error: %v", content.Name, err)
	}
	snap, err = snapCli.VolumesnapshotV1alpha1().VolumeSnapshots(namespace).Create(snap)
	if err != nil {
		return errors.Errorf("Failed to create content, Volumesnapshot: %s, Error: %v", snap.Name, err)
	}
	if !waitForReady {
		return nil
	}

	return WaitOnReadyToUse(ctx, snapCli, snap.Name, snap.Namespace)
}

// WaitOnReadyToUse will block until the Volumesnapshot in namespace 'namespace' with name 'snapshotName'
// has status 'ReadyToUse' or 'ctx.Done()' is signalled.
func WaitOnReadyToUse(ctx context.Context, snapCli snapshotclient.Interface, snapshotName, namespace string) error {
	return poll.Wait(ctx, func(context.Context) (bool, error) {
		snap, err := snapCli.VolumesnapshotV1alpha1().VolumeSnapshots(namespace).Get(snapshotName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return snap.Status.ReadyToUse, nil
	})
}

func getContent(ctx context.Context, snapCli snapshotclient.Interface, contentName string) (*snapshot.VolumeSnapshotContent, error) {
	return snapCli.VolumesnapshotV1alpha1().VolumeSnapshotContents().Get(contentName, metav1.GetOptions{})
}
