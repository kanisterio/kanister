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

package function

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	awsconfig "github.com/kanisterio/kanister/pkg/aws"
	"github.com/kanisterio/kanister/pkg/blockstorage"
	"github.com/kanisterio/kanister/pkg/blockstorage/awsebs"
	"github.com/kanisterio/kanister/pkg/blockstorage/getter"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/secrets"
	"github.com/kanisterio/kanister/pkg/utils"
)

func init() {
	_ = kanister.Register(&createVolumeSnapshotFunc{})
}

var (
	_ kanister.Func = (*createVolumeSnapshotFunc)(nil)
)

const (
	// CreateVolumeSnapshotFuncName gives the name of the function
	CreateVolumeSnapshotFuncName     = "CreateVolumeSnapshot"
	CreateVolumeSnapshotNamespaceArg = "namespace"
	CreateVolumeSnapshotPVCsArg      = "pvcs"
	CreateVolumeSnapshotSkipWaitArg  = "skipWait"
)

type createVolumeSnapshotFunc struct {
	progressPercent string
}

func (*createVolumeSnapshotFunc) Name() string {
	return CreateVolumeSnapshotFuncName
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

func ValidateLocationForBlockstorage(profile *param.Profile, sType blockstorage.Type) error {
	if err := ValidateProfile(profile); err != nil {
		return errors.Wrapf(err, "Profile Validation failed")
	}
	switch sType {
	case blockstorage.TypeEBS:
		if profile.Location.Type != crv1alpha1.LocationTypeS3Compliant {
			return errors.Errorf("Location type %s not supported for blockstorage type %s", profile.Location.Type, sType)
		}
		if len(profile.Location.Region) == 0 {
			return errors.Errorf("Region is not set. Required for blockstorage type %s", sType)
		}
	case blockstorage.TypeGPD:
		if profile.Location.Type != crv1alpha1.LocationTypeGCS {
			return errors.Errorf("Location type %s not supported for blockstorage type %s", profile.Location.Type, sType)
		}
	default:
		return errors.Errorf("Storage provider not supported %s", sType)
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
			volSnapInfo, err := snapshotVolume(ctx, volInfo, skipWait)
			if err != nil {
				errstrings = append(errstrings, err.Error())
			} else {
				PVCData = append(PVCData, *volSnapInfo)
			}
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

func snapshotVolume(ctx context.Context, volume volumeInfo, skipWait bool) (*VolumeSnapshotInfo, error) {
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
	pvc, err := kubeCli.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get PVC, PVC name: %s, namespace: %s", name, namespace)
	}
	pvName := pvc.Spec.VolumeName
	if pvName == "" {
		return nil, errors.Errorf("PVC %s in namespace %s not bound", name, namespace)
	}
	pv, err := kubeCli.CoreV1().PersistentVolumes().Get(ctx, pvName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get PV %s, namespace: %s", pvName, namespace)
	}
	pvLabels := pv.GetObjectMeta().GetLabels()
	var size int64
	if cap, ok := pv.Spec.Capacity[corev1.ResourceStorage]; ok {
		size = cap.Value()
	}
	// Check to see which provider is the source. Spec mandates only one of the provider
	// fields will be set
	switch {
	case pv.Spec.AWSElasticBlockStore != nil:
		ebs := pv.Spec.AWSElasticBlockStore
		if err = ValidateLocationForBlockstorage(tp.Profile, blockstorage.TypeEBS); err != nil {
			return nil, errors.Wrap(err, "Profile validation failed")
		}
		// Get Region from PV label or EC2 metadata
		if pvRegion := kube.GetRegionFromLabels(pvLabels); pvRegion != "" {
			region = pvRegion
		} else {
			region, err = awsebs.GetRegionFromEC2Metadata()
			if err != nil {
				return nil, err
			}
		}
		if pvZone := kube.GetZoneFromLabels(pvLabels); pvZone != "" {
			config := getConfig(tp.Profile, blockstorage.TypeEBS)
			config[awsconfig.ConfigRegion] = region
			provider, err = getter.Get(blockstorage.TypeEBS, config)
			if err != nil {
				return nil, errors.Wrap(err, "Could not get storage provider")
			}
			return &volumeInfo{provider: provider, volumeID: filepath.Base(ebs.VolumeID), sType: blockstorage.TypeEBS, volZone: pvZone, pvc: name, size: size, region: region}, nil
		}
		return nil, errors.Errorf("PV zone label is empty, pvName: %s, namespace: %s", pvName, namespace)

	case pv.Spec.GCEPersistentDisk != nil:
		gpd := pv.Spec.GCEPersistentDisk
		region = ""
		if err = ValidateLocationForBlockstorage(tp.Profile, blockstorage.TypeGPD); err != nil {
			return nil, errors.Wrap(err, "Profile validation failed")
		}
		if pvZone := kube.GetZoneFromLabels(pvLabels); pvZone != "" {
			config := getConfig(tp.Profile, blockstorage.TypeGPD)
			provider, err = getter.Get(blockstorage.TypeGPD, config)
			if err != nil {
				return nil, errors.Wrap(err, "Could not get storage provider")
			}
			return &volumeInfo{provider: provider, volumeID: filepath.Base(gpd.PDName), sType: blockstorage.TypeGPD, volZone: pvZone, pvc: name, size: size, region: region}, nil
		}
		return nil, errors.Errorf("PV zone label is empty, pvName: %s, namespace: %s", pvName, namespace)
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

func (c *createVolumeSnapshotFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	// Set progress percent
	c.progressPercent = progress.StartedPercent
	defer func() { c.progressPercent = progress.CompletedPercent }()

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
func getConfig(profile *param.Profile, sType blockstorage.Type) map[string]string {
	config := make(map[string]string)
	switch sType {
	case blockstorage.TypeEBS:
		if profile.Credential.Type == param.CredentialTypeKeyPair {
			config[awsconfig.AccessKeyID] = profile.Credential.KeyPair.ID
			config[awsconfig.SecretAccessKey] = profile.Credential.KeyPair.Secret
		} else if profile.Credential.Type == param.CredentialTypeSecret {
			config[awsconfig.AccessKeyID] = string(profile.Credential.Secret.Data[secrets.AWSAccessKeyID])
			config[awsconfig.SecretAccessKey] = string(profile.Credential.Secret.Data[secrets.AWSSecretAccessKey])
			config[awsconfig.ConfigRole] = string(profile.Credential.Secret.Data[secrets.ConfigRole])
		}
	case blockstorage.TypeGPD:
		config[blockstorage.GoogleProjectID] = profile.Credential.KeyPair.ID
		config[blockstorage.GoogleServiceKey] = profile.Credential.KeyPair.Secret
	}
	return config
}

func (*createVolumeSnapshotFunc) RequiredArgs() []string {
	return []string{CreateVolumeSnapshotNamespaceArg}
}

func (*createVolumeSnapshotFunc) Arguments() []string {
	return []string{
		CreateVolumeSnapshotNamespaceArg,
		CreateVolumeSnapshotPVCsArg,
		CreateVolumeSnapshotSkipWaitArg,
	}
}

func (c *createVolumeSnapshotFunc) Validate(args map[string]any) error {
	if err := utils.CheckSupportedArgs(c.Arguments(), args); err != nil {
		return err
	}

	return utils.CheckRequiredArgs(c.RequiredArgs(), args)
}

func (c *createVolumeSnapshotFunc) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    c.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}
