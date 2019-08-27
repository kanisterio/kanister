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

	. "gopkg.in/check.v1"
	"k8s.io/client-go/kubernetes/fake"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/blockstorage"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/testutil/mockblockstorage"
)

type DeleteVolumeSnapshotTestSuite struct{}

var _ = Suite(&DeleteVolumeSnapshotTestSuite{})

func (s *DeleteVolumeSnapshotTestSuite) TestDeleteVolumeSnapshot(c *C) {
	ctx := context.Background()
	ns := "ns"
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
	cli := fake.NewSimpleClientset()

	tags := []*blockstorage.KeyValue{
		{Key: "testkey", Value: "testval"},
	}
	volInfo1 := VolumeSnapshotInfo{SnapshotID: "snap-1", Type: blockstorage.TypeEBS, Region: "us-west-2", PVCName: "pvc-1", Az: "us-west-2a", Tags: tags, VolumeType: "ssd"}
	volInfo2 := VolumeSnapshotInfo{SnapshotID: "snap-2", Type: blockstorage.TypeEBS, Region: "us-west-2", PVCName: "pvc-2", Az: "us-west-2a", Tags: tags, VolumeType: "ssd"}
	var PVCData1 []VolumeSnapshotInfo
	PVCData1 = append(PVCData1, volInfo1)
	PVCData1 = append(PVCData1, volInfo2)
	info, err := json.Marshal(PVCData1)
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
		providerList, err := deleteVolumeSnapshot(ctx, cli, ns, tc.snapshotinfo, profile, mockGetter)
		c.Assert(providerList, Not(Equals), tc.check)
		c.Assert(err, tc.check)
		if err != nil {
			continue
		}
		c.Assert(len(providerList) == 2, Equals, true)
		provider, ok := providerList["pvc-1"]
		c.Assert(ok, Equals, true)
		c.Assert(len(provider.(*mockblockstorage.Provider).SnapIDList) == 1, Equals, true)
		c.Assert(mockblockstorage.CheckID("snap-1", provider.(*mockblockstorage.Provider).SnapIDList), Equals, true)
		c.Assert(len(provider.(*mockblockstorage.Provider).DeletedSnapIDList) == 1, Equals, true)
		c.Assert(mockblockstorage.CheckID("snap-1", provider.(*mockblockstorage.Provider).DeletedSnapIDList), Equals, true)

		provider, ok = providerList["pvc-2"]
		c.Assert(ok, Equals, true)
		c.Assert(len(provider.(*mockblockstorage.Provider).SnapIDList) == 1, Equals, true)
		c.Assert(mockblockstorage.CheckID("snap-2", provider.(*mockblockstorage.Provider).SnapIDList), Equals, true)
		c.Assert(len(provider.(*mockblockstorage.Provider).DeletedSnapIDList) == 1, Equals, true)
		c.Assert(mockblockstorage.CheckID("snap-2", provider.(*mockblockstorage.Provider).DeletedSnapIDList), Equals, true)
	}
}
