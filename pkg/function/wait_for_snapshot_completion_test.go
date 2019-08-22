// Copyright 2019 Kasten Inc.
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

	. "gopkg.in/check.v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/blockstorage"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/testutil/mockblockstorage"
)

type WaitForSnapshotCompletionTestSuite struct{}

var _ = Suite(&WaitForSnapshotCompletionTestSuite{})

func (s *WaitForSnapshotCompletionTestSuite) TestWait(c *C) {
	ctx := context.Background()
	mockGetter := mockblockstorage.NewGetter()
	profile := &param.Profile{
		Location: crv1alpha1.Location{
			Type:   crv1alpha1.LocationTypeS3Compliant,
			Region: "us-west-2",
		},
		Credential: param.Credential{
			Type: param.CredentialTypeKeyPair,
			KeyPair: &param.KeyPair{
				ID:     "foo",
				Secret: "bar",
			},
		},
	}
	pvcData1 := []VolumeSnapshotInfo{
		VolumeSnapshotInfo{SnapshotID: "snap-1", Type: blockstorage.TypeEBS, Region: "us-west-2", PVCName: "pvc-1", Az: "us-west-2a", VolumeType: "ssd"},
		VolumeSnapshotInfo{SnapshotID: "snap-2", Type: blockstorage.TypeEBS, Region: "us-west-2", PVCName: "pvc-2", Az: "us-west-2a", VolumeType: "ssd"},
	}
	info, err := json.Marshal(pvcData1)
	c.Assert(err, IsNil)
	snapinfo := string(info)
	for _, tc := range []struct {
		snapshotinfo string
		check        Checker
	}{
		{
			snapshotinfo: snapinfo,
			check:        IsNil,
		},
	} {
		err := waitForSnapshotsCompletion(ctx, tc.snapshotinfo, profile, mockGetter)
		c.Assert(err, tc.check)
	}
}
