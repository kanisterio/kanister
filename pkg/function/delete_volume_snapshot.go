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
	"strings"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	awsconfig "github.com/kanisterio/kanister/pkg/aws"
	"github.com/kanisterio/kanister/pkg/blockstorage"
	"github.com/kanisterio/kanister/pkg/blockstorage/getter"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/utils"
)

func init() {
	_ = kanister.Register(&deleteVolumeSnapshotFunc{})
}

var (
	_ kanister.Func = (*deleteVolumeSnapshotFunc)(nil)
)

const (
	// DeleteVolumeSnapshotFuncName gives the function name
	DeleteVolumeSnapshotFuncName     = "DeleteVolumeSnapshot"
	DeleteVolumeSnapshotNamespaceArg = "namespace"
	DeleteVolumeSnapshotManifestArg  = "snapshots"
)

type deleteVolumeSnapshotFunc struct {
	progressPercent string
}

func (*deleteVolumeSnapshotFunc) Name() string {
	return DeleteVolumeSnapshotFuncName
}

//nolint:unparam
func deleteVolumeSnapshot(ctx context.Context, cli kubernetes.Interface, namespace, snapshotinfo string, profile *param.Profile, getter getter.Getter) (map[string]blockstorage.Provider, error) {
	PVCData := []VolumeSnapshotInfo{}
	err := json.Unmarshal([]byte(snapshotinfo), &PVCData)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not decode JSON data")
	}
	// providerList required for unit testing
	providerList := make(map[string]blockstorage.Provider)
	for _, pvcInfo := range PVCData {
		if err = ValidateLocationForBlockstorage(profile, pvcInfo.Type); err != nil {
			return nil, errors.Wrap(err, "Profile validation failed")
		}
		config := getConfig(profile, pvcInfo.Type)
		if pvcInfo.Type == blockstorage.TypeEBS {
			config[awsconfig.ConfigRegion] = pvcInfo.Region
		}

		provider, err := getter.Get(pvcInfo.Type, config)
		if err != nil {
			return nil, errors.Wrapf(err, "Could not get storage provider")
		}
		snapshot, err := provider.SnapshotGet(ctx, pvcInfo.SnapshotID)
		if err != nil {
			if strings.Contains(err.Error(), blockstorage.SnapshotDoesNotExistError) {
				log.WithContext(ctx).Print("Snapshot already deleted", field.M{"SnapshotID": pvcInfo.SnapshotID})
			} else {
				return nil, errors.Wrapf(err, "Failed to get Snapshot from Provider")
			}
		}
		if err = provider.SnapshotDelete(ctx, snapshot); err != nil {
			return nil, err
		}
		log.WithContext(ctx).Print("Successfully deleted snapshot", field.M{"SnapshotID": pvcInfo.SnapshotID})
		providerList[pvcInfo.PVCName] = provider
	}
	return providerList, nil
}

func (d *deleteVolumeSnapshotFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	// Set progress percent
	d.progressPercent = progress.StartedPercent
	defer func() { d.progressPercent = progress.CompletedPercent }()

	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	var namespace, snapshotinfo string
	if err = Arg(args, DeleteVolumeSnapshotNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, DeleteVolumeSnapshotManifestArg, &snapshotinfo); err != nil {
		return nil, err
	}
	_, err = deleteVolumeSnapshot(ctx, cli, namespace, snapshotinfo, tp.Profile, getter.New())
	return nil, err
}

func (*deleteVolumeSnapshotFunc) RequiredArgs() []string {
	return []string{
		DeleteVolumeSnapshotNamespaceArg,
		DeleteVolumeSnapshotManifestArg,
	}
}

func (*deleteVolumeSnapshotFunc) Arguments() []string {
	return []string{
		DeleteVolumeSnapshotNamespaceArg,
		DeleteVolumeSnapshotManifestArg,
	}
}

func (d *deleteVolumeSnapshotFunc) Validate(args map[string]any) error {
	if err := utils.CheckSupportedArgs(d.Arguments(), args); err != nil {
		return err
	}

	return utils.CheckRequiredArgs(d.RequiredArgs(), args)
}

func (d *deleteVolumeSnapshotFunc) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    d.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}
