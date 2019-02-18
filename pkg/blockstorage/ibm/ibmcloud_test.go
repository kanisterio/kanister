package ibm

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	ibmprov "github.com/IBM/ibmcloud-storage-volume-lib/lib/provider"
	. "gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/blockstorage"
)

const (
	testTagKey   = "kanister.io/testid"
	testTagValue = "unittest"
)

type TestIBMCloudBlock struct {
	provider blockstorage.Provider
	cli      *client
	testVol  *blockstorage.Volume
}

var _ = Suite(&TestIBMCloudBlock{})

func (s *TestIBMCloudBlock) SetUpSuite(c *C) {
	var apiKey string
	if apiK, ok := os.LookupEnv(IBMApiKeyEnv); ok {
		apiKey = apiK
	} else {
		c.Skip(fmt.Sprintf("Could not find env var %s with API key", IBMApiKeyEnv))
	}
	args := map[string]string{APIKeyArgName: apiKey}
	var err error
	ctx := context.Background()
	s.provider, err = NewProvider(ctx, args)
	c.Assert(err, IsNil)
	c.Assert(s.provider, NotNil)
	s.cli, err = newClient(context.Background(), args)
	c.Assert(err, IsNil)
	c.Assert(s.cli, NotNil)
	tmpVol := &blockstorage.Volume{
		Attributes: make(map[string]string),
	}

	tmpVol.VolumeType = string(s.cli.Service.Type())
	provT := "endurance"
	tmpVol.Attributes[ProviderTypeAttName] = string(ibmprov.VolumeProviderType(provT))
	tmpVol.Attributes[TierAttName] = "2"
	tmpVol.Size = 20
	tmpVol.Tags = []*blockstorage.KeyValue{
		{Key: testTagKey, Value: testTagValue},
		{Key: "kanister.io/testname", Value: c.TestName()},
	}
	tmpVol.Az = s.cli.SLCfg.SoftlayerDataCenter
	s.testVol, err = s.provider.VolumeCreate(ctx, *tmpVol)
	c.Assert(err, IsNil)
	c.Assert(s.testVol.ID, NotNil)
}

func (s TestIBMCloudBlock) TearDownSuite(c *C) {
	// Check whether or not the test volume was initialized
	c.Assert(s.testVol, NotNil)
	bsVol, err := s.provider.VolumeGet(context.Background(), s.testVol.ID, "")
	c.Assert(err, IsNil)
	err = s.provider.VolumeDelete(context.Background(), bsVol)
	c.Assert(err, IsNil)
}

func (s TestIBMCloudBlock) TestSnapshotCreate(c *C) {
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

func (s TestIBMCloudBlock) TestVolRestore(c *C) {
	bsVol, err := s.provider.VolumeGet(context.Background(), s.testVol.ID, "")
	c.Assert(err, IsNil)
	snapTTags := map[string]string{"ibmblock_unit_test_snap": fmt.Sprintf("test-snap-%d", time.Now().Unix())}
	bsSnap, err := s.provider.SnapshotCreate(context.Background(), *bsVol, snapTTags)
	defer s.provider.SnapshotDelete(context.Background(), bsSnap) // nolint: errcheck
	c.Assert(err, IsNil)
	tTags := map[string]string{"ibmblock_unit_test_restore_vol": fmt.Sprintf("test-vol-%d", time.Now().Unix())}
	resVol, err := s.provider.VolumeCreateFromSnapshot(context.Background(), *bsSnap, tTags)
	c.Assert(err, IsNil)
	cVol, err := s.cli.Service.VolumeGet(resVol.ID)
	c.Assert(err, IsNil)
	c.Assert(cVol, NotNil)
	err = s.provider.VolumeDelete(context.Background(), resVol)
	c.Assert(err, IsNil)
}
