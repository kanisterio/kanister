package function

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
)

func init() {
	kanister.Register(&createVolumeSnapshotFunc{})
}

var (
	_ kanister.Func = (*createVolumeSnapshotFunc)(nil)
)

const (
	CreateVolumeSnapshotNamespaceArg = "namespace"
	CreateVolumeSnapshotPVCsArg      = "pvcs"
)

type createVolumeSnapshotFunc struct{}

func (*createVolumeSnapshotFunc) Name() string {
	return "CreateVolumeSnapshot"
}

type VolumeSnapshotInfo struct {
	SnapshotID  string
	StorageType string
	Region      string
}

type volumeInfo struct {
	provider    string
	volumeID    string
	storageType string
	volZone     string
	pvc         string
	size        int64
}

func createVolumeSnapshot(ctx context.Context, tp param.TemplateParams, cli kubernetes.Interface, namespace string, pvcs []string) (map[string]interface{}, error) {

	PVCData := make([]VolumeSnapshotInfo, 0, len(pvcs))
	for _, pvc := range pvcs {
		volInfo, err := getPVCInfo(cli, namespace, pvc)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to get PVC info")
		}
		volSnapInfo, err := snapshotVolume(ctx, cli, volInfo, namespace)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to snapshot volume")
		}
		PVCData = append(PVCData, *volSnapInfo)
	}
	manifestData, err := json.Marshal(PVCData)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to encode JSON data")
	}

	return map[string]interface{}{"volumeSnapshotInfo": string(manifestData)}, nil
}

func snapshotVolume(ctx context.Context, cli kubernetes.Interface, vol *volumeInfo, namespace string) (*VolumeSnapshotInfo, error) {
	return &VolumeSnapshotInfo{SnapshotID: vol.volumeID, StorageType: vol.storageType, Region: ""}, nil
}

func getPVCInfo(kubeCli kubernetes.Interface, namespace string, name string) (*volumeInfo, error) {
	pvc, err := kubeCli.Core().PersistentVolumeClaims(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get PVC, PVC name: %s, namespace: %s", name, namespace)
	}
	pvName := pvc.Spec.VolumeName
	if pvName == "" {
		return nil, errors.Errorf("PVC %s in namespace %s not bound", name, namespace)
	}
	pv, err := kubeCli.Core().PersistentVolumes().Get(pvName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get PV %s, namespace: %s", pvName, namespace)
	}
	var size int64
	if cap, ok := pv.Spec.Capacity[v1.ResourceStorage]; ok {
		size = cap.Value()
	}
	// Check to see which provider is the source. Spec mandates only one of the provider
	// fields will be set
	if ebs := pv.Spec.AWSElasticBlockStore; ebs != nil {
		return &volumeInfo{provider: "EBS", volumeID: pvName, storageType: "EBS", volZone: "", pvc: name, size: size}, nil
	}
	if gpd := pv.Spec.GCEPersistentDisk; gpd != nil {
		return &volumeInfo{provider: "GPD", volumeID: pvName, storageType: "GPD", volZone: "", pvc: name, size: size}, nil
	}
	if ad := pv.Spec.AzureDisk; ad != nil {
		return &volumeInfo{provider: "AD", volumeID: pvName, storageType: "AD", volZone: "", pvc: name, size: size}, nil
	}
	if cinder := pv.Spec.Cinder; cinder != nil {
		return &volumeInfo{provider: "Cinder", volumeID: pvName, storageType: "Cinder", volZone: "", pvc: name, size: size}, nil
	}
	if ceph := pv.Spec.RBD; ceph != nil {
		return &volumeInfo{provider: "Ceph", volumeID: pvName, storageType: "Ceph", volZone: "", pvc: name, size: size}, nil
	}
	return nil, errors.New("Storage type not supported!")
}

func getPVCList(tp param.TemplateParams) ([]string, error) {
	var pvcList []string
	var podsToPvcs map[string]map[string]string
	switch {
	case tp.Deployment != nil:
		podsToPvcs = tp.Deployment.PersistentVolumeClaims
	case tp.StatefulSet != nil:
		podsToPvcs = tp.StatefulSet.PersistentVolumeClaims
	default:
		return nil, errors.New("Failed to get volumes")
	}
	for _, podToPvcs := range podsToPvcs {
		for pvc := range podToPvcs {
			pvcList = append(pvcList, pvc)
		}
	}
	if len(pvcList) == 0 {
		return nil, errors.New("No pvcs found")
	}
	return pvcList, nil
}

func (kef *createVolumeSnapshotFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	var namespace string
	var pvcs []string
	if err = Arg(args, CreateVolumeSnapshotNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err = OptArg(args, RestoreDataVolsArg, &pvcs, nil); err != nil {
		return nil, err
	}
	if len(pvcs) == 0 {
		// Fetch Volumes
		pvcs, err = getPVCList(tp)
		if err != nil {
			return nil, err
		}
	}
	return createVolumeSnapshot(ctx, tp, cli, namespace, pvcs)
}

func (*createVolumeSnapshotFunc) RequiredArgs() []string {
	return []string{CreateVolumeSnapshotNamespaceArg}
}
