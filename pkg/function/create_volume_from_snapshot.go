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
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	awsconfig "github.com/kanisterio/kanister/pkg/aws"
	"github.com/kanisterio/kanister/pkg/blockstorage"
	"github.com/kanisterio/kanister/pkg/blockstorage/getter"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/kube"
	kubevolume "github.com/kanisterio/kanister/pkg/kube/volume"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/utils"
)

func init() {
	_ = kanister.Register(&createVolumeFromSnapshotFunc{})
}

var (
	_ kanister.Func = (*createVolumeFromSnapshotFunc)(nil)
)

const (
	// CreateVolumeFromSnapshotFuncName gives the name of the function
	CreateVolumeFromSnapshotFuncName     = "CreateVolumeFromSnapshot"
	CreateVolumeFromSnapshotNamespaceArg = "namespace"
	CreateVolumeFromSnapshotManifestArg  = "snapshots"
	CreateVolumeFromSnapshotPVCNamesArg  = "pvcNames"
)

type createVolumeFromSnapshotFunc struct {
	progressPercent string
}

func (*createVolumeFromSnapshotFunc) Name() string {
	return CreateVolumeFromSnapshotFuncName
}

func createVolumeFromSnapshot(ctx context.Context, cli kubernetes.Interface, namespace, snapshotinfo string, pvcNames []string, profile *param.Profile, getter getter.Getter) (map[string]blockstorage.Provider, error) {
	PVCData := []VolumeSnapshotInfo{}
	err := json.Unmarshal([]byte(snapshotinfo), &PVCData)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not decode JSON data")
	}
	if len(pvcNames) > 0 && len(pvcNames) != len(PVCData) {
		return nil, errors.New("Invalid number of PVC names provided")
	}
	// providerList required for unit testing
	providerList := make(map[string]blockstorage.Provider)
	for i, pvcInfo := range PVCData {
		pvcName := pvcInfo.PVCName
		if len(pvcNames) > 0 {
			pvcName = pvcNames[i]
		}
		if err = ValidateLocationForBlockstorage(profile, pvcInfo.Type); err != nil {
			return nil, errors.Wrap(err, "Profile validation failed")
		}
		config := getConfig(profile, pvcInfo.Type)
		if pvcInfo.Type == blockstorage.TypeEBS {
			config[awsconfig.ConfigRegion] = pvcInfo.Region
		}

		provider, err := getter.Get(pvcInfo.Type, config)
		if err != nil {
			return nil, errors.Wrapf(err, "Could not get storage provider %v", pvcInfo.Type)
		}
		_, err = cli.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, pvcName, metav1.GetOptions{})
		if err == nil {
			if err = kubevolume.DeletePVC(cli, namespace, pvcName); err != nil {
				return nil, err
			}
		}
		snapshot, err := provider.SnapshotGet(ctx, pvcInfo.SnapshotID)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to get Snapshot from Provider")
		}

		tags := map[string]string{
			"pvcname": pvcName,
		}
		snapshot.Volume.VolumeType = pvcInfo.VolumeType
		snapshot.Volume.Az = pvcInfo.Az
		snapshot.Volume.Tags = pvcInfo.Tags
		vol, err := provider.VolumeCreateFromSnapshot(ctx, *snapshot, tags)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to create volume from snapshot, snapID: %s", snapshot.ID)
		}

		annotations := map[string]string{}
		pvc, err := kubevolume.CreatePVC(ctx, cli, namespace, pvcName, vol.SizeInBytes, vol.ID, annotations, nil, nil)
		if err != nil {
			return nil, errors.Wrapf(err, "Unable to create PVC for volume %v", *vol)
		}

		pvAnnotations := addPVProvisionedByAnnotation(nil, provider)
		pv, err := kubevolume.CreatePV(ctx, cli, vol, vol.Type, pvAnnotations, nil, nil)
		if err != nil {
			return nil, errors.Wrapf(err, "Unable to create PV for volume %v", *vol)
		}
		log.WithContext(ctx).Print("Restore/Create volume from snapshot completed", field.M{"PVC": pvc, "Volume": pv})
		providerList[pvcInfo.PVCName] = provider
	}
	return providerList, nil
}

func addPVProvisionedByAnnotation(annotations map[string]string, provider blockstorage.Provider) map[string]string {
	if annotations == nil {
		annotations = make(map[string]string)
	}

	storageType := provider.Type()
	switch storageType {
	case blockstorage.TypeGPD:
		annotations[consts.PVProvisionedByAnnotation] = consts.GCEPDProvisionerInTree
	case blockstorage.TypeEBS:
		annotations[consts.PVProvisionedByAnnotation] = consts.AWSEBSProvisionerInTree
	}

	return annotations
}

func (c *createVolumeFromSnapshotFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	// Set progress percent
	c.progressPercent = progress.StartedPercent
	defer func() { c.progressPercent = progress.CompletedPercent }()

	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	var namespace, snapshotinfo string
	var pvcNames []string
	if err = Arg(args, CreateVolumeFromSnapshotNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, CreateVolumeFromSnapshotManifestArg, &snapshotinfo); err != nil {
		return nil, err
	}
	if err = OptArg(args, CreateVolumeFromSnapshotPVCNamesArg, &pvcNames, nil); err != nil {
		return nil, err
	}
	_, err = createVolumeFromSnapshot(ctx, cli, namespace, snapshotinfo, pvcNames, tp.Profile, getter.New())
	return nil, err
}

func (*createVolumeFromSnapshotFunc) RequiredArgs() []string {
	return []string{
		CreateVolumeFromSnapshotNamespaceArg,
		CreateVolumeFromSnapshotManifestArg,
	}
}

func (*createVolumeFromSnapshotFunc) Arguments() []string {
	return []string{
		CreateVolumeFromSnapshotNamespaceArg,
		CreateVolumeFromSnapshotManifestArg,
		CreateVolumeFromSnapshotPVCNamesArg,
	}
}

func (c *createVolumeFromSnapshotFunc) Validate(args map[string]any) error {
	if err := utils.CheckSupportedArgs(c.Arguments(), args); err != nil {
		return err
	}

	return utils.CheckRequiredArgs(c.RequiredArgs(), args)
}

func (crs *createVolumeFromSnapshotFunc) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    crs.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}
