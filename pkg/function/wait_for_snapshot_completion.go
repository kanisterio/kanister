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

	"github.com/pkg/errors"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/blockstorage"
	"github.com/kanisterio/kanister/pkg/blockstorage/getter"
	awsconfig "github.com/kanisterio/kanister/pkg/config/aws"
	"github.com/kanisterio/kanister/pkg/param"
)

func init() {
	kanister.Register(&waitForSnapshotCompletionFunc{})
}

var (
	_ kanister.Func = (*waitForSnapshotCompletionFunc)(nil)
)

const (
	// WaitForSnapshotCompletionFuncName gives the function name
	WaitForSnapshotCompletionFuncName     = "WaitForSnapshotCompletion"
	WaitForSnapshotCompletionSnapshotsArg = "snapshots"
)

type waitForSnapshotCompletionFunc struct{}

func (*waitForSnapshotCompletionFunc) Name() string {
	return WaitForSnapshotCompletionFuncName
}

func (*waitForSnapshotCompletionFunc) RequiredArgs() []string {
	return []string{WaitForSnapshotCompletionSnapshotsArg}
}

func (kef *waitForSnapshotCompletionFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var snapshotinfo string
	if err := Arg(args, WaitForSnapshotCompletionSnapshotsArg, &snapshotinfo); err != nil {
		return nil, err
	}
	return nil, waitForSnapshotsCompletion(ctx, snapshotinfo, tp.Profile, getter.New())
}

func waitForSnapshotsCompletion(ctx context.Context, snapshotinfo string, profile *param.Profile, getter getter.Getter) error {
	PVCData := []VolumeSnapshotInfo{}
	err := json.Unmarshal([]byte(snapshotinfo), &PVCData)
	if err != nil {
		return errors.Wrapf(err, "Could not decode JSON data")
	}
	for _, pvcInfo := range PVCData {
		config := make(map[string]string)
		if err = ValidateLocationForBlockstorage(profile, pvcInfo.Type); err != nil {
			return errors.Wrap(err, "Profile validation failed")
		}
		switch pvcInfo.Type {
		case blockstorage.TypeEBS:
			config[awsconfig.ConfigRegion] = pvcInfo.Region
			config[awsconfig.AccessKeyID] = profile.Credential.KeyPair.ID
			config[awsconfig.SecretAccessKey] = profile.Credential.KeyPair.Secret
		case blockstorage.TypeGPD:
			config[blockstorage.GoogleProjectID] = profile.Credential.KeyPair.ID
			config[blockstorage.GoogleServiceKey] = profile.Credential.KeyPair.Secret
		default:
			return errors.New("Storage provider not supported " + string(pvcInfo.Type))
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
