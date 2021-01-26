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

// +build !unit

package ibm

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	ibmprov "github.com/IBM/ibmcloud-storage-volume-lib/lib/provider"
	"github.com/luci/go-render/render"
	. "gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/blockstorage"
)

const (
	testTagKey   = "kanister.io/testid"
	testTagValue = "unittest"
)

type TestIBMCloud struct {
	provider      blockstorage.Provider
	cli           *client
	testVol       *blockstorage.Volume
	softlayerFile bool
	volAtts       map[string]string
}

//These are not executed as part of Pipeline, but usefull for development
var softlayerVolAtts = map[string]string{
	ProviderTypeAttName: string(ibmprov.VolumeProviderType("endurance")),
	TierAttName:         "2",
}

var _ = Suite(&TestIBMCloud{softlayerFile: false, volAtts: softlayerVolAtts})
var _ = Suite(&TestIBMCloud{softlayerFile: true, volAtts: softlayerVolAtts})

func (s *TestIBMCloud) SetUpSuite(c *C) {
	c.Skip("IBM tests are too flaky to run in CI")
	var apiKey string
	if apiK, ok := os.LookupEnv(IBMApiKeyEnv); ok {
		apiKey = apiK
	} else {
		c.Skip(fmt.Sprintf("Could not find env var %s with API key", IBMApiKeyEnv))
	}
	args := map[string]string{APIKeyArgName: apiKey}
	if s.softlayerFile {
		args[SoftlayerFileAttName] = "true"
	}

	var err error
	ctx := context.Background()
	s.provider, err = NewProvider(ctx, args)
	c.Assert(err, IsNil)
	c.Assert(s.provider, NotNil)
	s.cli, err = newClient(context.Background(), args)
	c.Assert(err, IsNil)
	c.Assert(s.cli, NotNil)
	tmpVol := &blockstorage.Volume{
		Attributes: s.volAtts,
	}
	tmpVol.VolumeType = string(s.cli.Service.Type())
	tmpVol.SizeInBytes = 20 * blockstorage.BytesInGi
	tmpVol.Tags = []*blockstorage.KeyValue{
		{Key: testTagKey, Value: testTagValue},
		{Key: "kanister.io/testname", Value: c.TestName()},
	}
	tmpVol.Az = s.cli.SLCfg.SoftlayerDataCenter
	s.testVol, err = s.provider.VolumeCreate(ctx, *tmpVol)
	c.Log(fmt.Sprintf("sl cfg %s", render.Render(softLayerCfg)))
	c.Assert(err, IsNil)
	c.Assert(s.testVol.ID, NotNil)
}

func (s TestIBMCloud) TearDownSuite(c *C) {
	c.Skip("IBM tests are too flaky to run in CI")
	// Check whether or not the test volume was initialized
	if _, ok := os.LookupEnv(IBMApiKeyEnv); !ok {
		c.Skip("Skipping TearDown")
	}
	c.Assert(s.testVol, NotNil)
	bsVol, err := s.provider.VolumeGet(context.Background(), s.testVol.ID, "")
	c.Assert(err, IsNil)
	err = s.provider.VolumeDelete(context.Background(), bsVol)
	c.Assert(err, IsNil)
}

func (s TestIBMCloud) TestSnapshotCreate(c *C) {
	bsVol, err := s.provider.VolumeGet(context.Background(), s.testVol.ID, "")
	c.Assert(err, IsNil)
	c.Assert(bsVol.ID, Equals, s.testVol.ID)
	snapTTags := map[string]string{"ibmblock_unit_test_snap": fmt.Sprintf("test-snap-%d", time.Now().Unix())}
	bsSnap, err := s.provider.SnapshotCreate(context.Background(), *bsVol, snapTTags)
	c.Assert(err, IsNil)
	c.Assert(bsSnap, NotNil)
	snaps4Vol, err := s.cli.Service.ListAllSnapshots(bsVol.ID)
	c.Assert(err, IsNil)
	c.Assert(len(snaps4Vol) > 0, Equals, true)
	var inCloud bool
	for _, tSnap := range snaps4Vol {
		if strings.Compare(tSnap.SnapshotID, bsSnap.ID) == 0 {
			inCloud = true
			break
		}
	}
	c.Assert(inCloud, Equals, true)
	err = s.provider.SnapshotDelete(context.Background(), bsSnap)
	c.Check(err, IsNil)
}

func (s TestIBMCloud) TestVolRestore(c *C) {
	bsVol, err := s.provider.VolumeGet(context.Background(), s.testVol.ID, "")
	c.Assert(err, IsNil)
	snapTTags := map[string]string{"ibmblock_unit_test_snap": fmt.Sprintf("test-snap-%d", time.Now().Unix())}
	bsSnap, err := s.provider.SnapshotCreate(context.Background(), *bsVol, snapTTags)
	defer s.provider.SnapshotDelete(context.Background(), bsSnap) // nolint: errcheck
	c.Assert(err, IsNil)
	tTags := map[string]string{"ibmblock_unit_test_restore_vol": fmt.Sprintf("test-vol-%d", time.Now().Unix())}
	resVol, err := s.provider.VolumeCreateFromSnapshot(context.Background(), *bsSnap, tTags)
	c.Assert(err, IsNil)
	cVol, err := s.cli.Service.GetVolume(resVol.ID)
	c.Assert(err, IsNil)
	c.Assert(cVol, NotNil)
	err = s.provider.VolumeDelete(context.Background(), resVol)
	c.Assert(err, IsNil)
}
