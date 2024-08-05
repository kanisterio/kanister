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

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	awsconfig "github.com/kanisterio/kanister/pkg/aws"
	"github.com/kanisterio/kanister/pkg/blockstorage"
	"github.com/kanisterio/kanister/pkg/blockstorage/getter"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/utils"
)

func init() {
	_ = kanister.Register(&waitForSnapshotCompletionFunc{})
}

var (
	_ kanister.Func = (*waitForSnapshotCompletionFunc)(nil)
)

const (
	// WaitForSnapshotCompletionFuncName gives the function name
	WaitForSnapshotCompletionFuncName     = "WaitForSnapshotCompletion"
	WaitForSnapshotCompletionSnapshotsArg = "snapshots"
)

type waitForSnapshotCompletionFunc struct {
	progressPercent string
}

func (*waitForSnapshotCompletionFunc) Name() string {
	return WaitForSnapshotCompletionFuncName
}

func (*waitForSnapshotCompletionFunc) RequiredArgs() []string {
	return []string{WaitForSnapshotCompletionSnapshotsArg}
}

func (*waitForSnapshotCompletionFunc) Arguments() []string {
	return []string{WaitForSnapshotCompletionSnapshotsArg}
}

func (w *waitForSnapshotCompletionFunc) Validate(args map[string]any) error {
	if err := utils.CheckSupportedArgs(w.Arguments(), args); err != nil {
		return err
	}

	return utils.CheckRequiredArgs(w.RequiredArgs(), args)
}

func (w *waitForSnapshotCompletionFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	// Set progress percent
	w.progressPercent = progress.StartedPercent
	defer func() { w.progressPercent = progress.CompletedPercent }()

	var snapshotinfo string
	if err := Arg(args, WaitForSnapshotCompletionSnapshotsArg, &snapshotinfo); err != nil {
		return nil, err
	}
	return nil, waitForSnapshotsCompletion(ctx, snapshotinfo, tp.Profile, getter.New())
}

func (w *waitForSnapshotCompletionFunc) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    w.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}

func waitForSnapshotsCompletion(ctx context.Context, snapshotinfo string, profile *param.Profile, getter getter.Getter) error {
	PVCData := []VolumeSnapshotInfo{}
	err := json.Unmarshal([]byte(snapshotinfo), &PVCData)
	if err != nil {
		return errors.Wrapf(err, "Could not decode JSON data")
	}

	for _, pvcInfo := range PVCData {
		if err = ValidateLocationForBlockstorage(profile, pvcInfo.Type); err != nil {
			return errors.Wrap(err, "Profile validation failed")
		}
		config := getConfig(profile, pvcInfo.Type)
		if pvcInfo.Type == blockstorage.TypeEBS {
			config[awsconfig.ConfigRegion] = pvcInfo.Region
		}

		provider, err := getter.Get(pvcInfo.Type, config)
		if err != nil {
			return errors.Wrapf(err, "Could not get storage provider %v", pvcInfo.Type)
		}
		snapshot, err := provider.SnapshotGet(ctx, pvcInfo.SnapshotID)
		if err != nil {
			return errors.Wrapf(err, "Failed to get Snapshot from Provider")
		}
		if err = provider.SnapshotCreateWaitForCompletion(ctx, snapshot); err != nil {
			return errors.Wrap(err, "Snapshot creation did not complete "+snapshot.ID)
		}
	}
	return nil
}
