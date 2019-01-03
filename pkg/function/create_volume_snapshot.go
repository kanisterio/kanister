package function

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/blockstorage"
	"github.com/kanisterio/kanister/pkg/blockstorage/awsebs"
	"github.com/kanisterio/kanister/pkg/blockstorage/getter"
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
	CreateVolumeSnapshotSkipWaitArg  = "skipWait"
)

type createVolumeSnapshotFunc struct{}

func (*createVolumeSnapshotFunc) Name() string {
	return "CreateVolumeSnapshot"
}

type VolumeSnapshotInfo struct {
	SnapshotID string
	Type       blockstorage.Type
	Region     string
	PVCName    string
	Az         string
	Tags       blockstorage.VolumeTags
	VolumeType string
}

type volumeInfo struct {
	provider blockstorage.Provider
	volumeID string
	sType    blockstorage.Type
	volZone  string
	pvc      string
	size     int64
	region   string
}

func ValidateProfile(profile *param.Profile) error {
	if profile == nil {
		return errors.New("Profile must be non-nil")
	}
	if profile.Location.Type != crv1alpha1.LocationTypeS3Compliant {
		return errors.New("Location type not supported")
	}
	if len(profile.Location.Region) == 0 {
		return errors.New("Region is not set")
	}
	if profile.Credential.Type != param.CredentialTypeKeyPair {
		return errors.New("Credential type not supported")
	}
	if len(profile.Credential.KeyPair.ID) == 0 {
		return errors.New("AWS access key id is not set")
	}
	if len(profile.Credential.KeyPair.Secret) == 0 {
		return errors.New("Secret access key is not set")
	}
	return nil
}

func createVolumeSnapshot(ctx context.Context, tp param.TemplateParams, cli kubernetes.Interface, namespace string, pvcs []string, getter getter.Getter, skipWait bool) (map[string]interface{}, error) {
	vols := make([]volumeInfo, 0, len(pvcs))
	for _, pvc := range pvcs {
		volInfo, err := getPVCInfo(ctx, cli, namespace, pvc, tp, getter)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to get PVC info")
		}
		vols = append(vols, *volInfo)
	}

	var PVCData []VolumeSnapshotInfo
	var wg sync.WaitGroup
	var errstrings []string
	for _, vol := range vols {
		wg.Add(1)
		go func(volInfo volumeInfo) {
			defer wg.Done()
			volSnapInfo, err := snapshotVolume(ctx, volInfo, namespace, skipWait)
			if err != nil {
				errstrings = append(errstrings, err.Error())
			} else {
				PVCData = append(PVCData, *volSnapInfo)
			}
			return
		}(vol)
	}
	wg.Wait()

	err := fmt.Errorf(strings.Join(errstrings, "\n"))
	if len(err.Error()) > 0 {
		return nil, errors.Wrapf(err, "Failed to snapshot one of the volumes")
	}

	manifestData, err := json.Marshal(PVCData)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to encode JSON data")
	}

	return map[string]interface{}{"volumeSnapshotInfo": string(manifestData)}, nil
}

func snapshotVolume(ctx context.Context, volume volumeInfo, namespace string, skipWait bool) (*VolumeSnapshotInfo, error) {
	provider := volume.provider
	vol, err := provider.VolumeGet(ctx, volume.volumeID, volume.volZone)
	if err != nil {
		return nil, errors.Wrapf(err, "Volume unavailable, volumeID: %s", volume.volumeID)
	}
	if vol.Encrypted {
		return nil, errors.New("Encrypted volumes are unsupported")
	}

	// Snapshot the volume.
	tags := map[string]string{
		"pvcname": volume.pvc,
	}
	if err = provider.SetTags(ctx, vol, tags); err != nil {
		return nil, err
	}
	snap, err := provider.SnapshotCreate(ctx, *vol, tags)
	if err != nil {
		return nil, err
	}
	if !skipWait {
		if err := provider.SnapshotCreateWaitForCompletion(ctx, snap); err != nil {
			return nil, errors.Wrap(err, "Snapshot creation did not complete")
		}
	}
	return &VolumeSnapshotInfo{SnapshotID: snap.ID, Type: volume.sType, Region: volume.region, PVCName: volume.pvc, Az: snap.Volume.Az, Tags: snap.Volume.Tags, VolumeType: snap.Volume.VolumeType}, nil
}

func getPVCInfo(ctx context.Context, kubeCli kubernetes.Interface, namespace string, name string, tp param.TemplateParams, getter getter.Getter) (*volumeInfo, error) {
	_ = ctx
	var region string
	var provider blockstorage.Provider
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
	pvLabels := pv.GetObjectMeta().GetLabels()
	var size int64
	if cap, ok := pv.Spec.Capacity[v1.ResourceStorage]; ok {
		size = cap.Value()
	}
	// Check to see which provider is the source. Spec mandates only one of the provider
	// fields will be set
	config := make(map[string]string)
	if ebs := pv.Spec.AWSElasticBlockStore; ebs != nil {
		if err = ValidateProfile(tp.Profile); err != nil {
			return nil, errors.Wrap(err, "Profile validation failed")
		}
		// Get Region from PV label or EC2 metadata
		if pvRegion, ok := pvLabels[kube.PVRegionLabelName]; ok {
			region = pvRegion
		} else {
			region, err = awsebs.GetRegionFromEC2Metadata()
			if err != nil {
				return nil, err
			}
		}
		if pvZone, ok := pvLabels[kube.PVZoneLabelName]; ok {
			config[awsebs.ConfigRegion] = region
			config[awsebs.AccessKeyID] = tp.Profile.Credential.KeyPair.ID
			config[awsebs.SecretAccessKey] = tp.Profile.Credential.KeyPair.Secret
			provider, err = getter.Get(blockstorage.TypeEBS, config)
			if err != nil {
				return nil, errors.Wrap(err, "Could not get storage provider")
			}
			return &volumeInfo{provider: provider, volumeID: filepath.Base(ebs.VolumeID), sType: blockstorage.TypeEBS, volZone: pvZone, pvc: name, size: size, region: region}, nil
		}
		return nil, errors.Errorf("PV zone label is empty, pvName: %s, namespace: %s", pvName, namespace)
	}
	return nil, errors.New("Storage type not supported")
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
	var skipWait bool
	if err = Arg(args, CreateVolumeSnapshotNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err = OptArg(args, CreateVolumeSnapshotPVCsArg, &pvcs, nil); err != nil {
		return nil, err
	}
	if err = OptArg(args, CreateVolumeSnapshotSkipWaitArg, &skipWait, nil); err != nil {
		return nil, err
	}
	if len(pvcs) == 0 {
		// Fetch Volumes
		pvcs, err = getPVCList(tp)
		if err != nil {
			return nil, err
		}
	}
	return createVolumeSnapshot(ctx, tp, cli, namespace, pvcs, getter.New(), skipWait)
}

func (*createVolumeSnapshotFunc) RequiredArgs() []string {
	return []string{CreateVolumeSnapshotNamespaceArg}
}
