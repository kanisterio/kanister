package vmware

import (
	"context"

	"github.com/gofrs/uuid"
	"gopkg.in/check.v1"
)

type VmwareManSuite struct{}

var _ = check.Suite(&VmwareManSuite{})

func (s *VmwareManSuite) TestCreateAndListSnapshots(c *check.C) {
	// c.Skip("manual testing")
	volumeID := "55c3e39b-95b0-40d1-aaed-ea11be829fa6"
	provider, _ := NewProvider(map[string]string{
		VSphereEndpointKey: "",
		VSphereUsernameKey: "",
		VSpherePasswordKey: "",
	})
	ftpProvider := provider.(*FcdProvider)

	ftpProvider.SetCategoryID("K10:0c66728a-dd0d-11ec-9939-ca6a7623d809")
	ctx := context.Background()

	guid1, _ := uuid.NewV1()
	guid2, _ := uuid.NewV1()
	tags := map[string]string{
		DescriptionTag:  guid1.String(),
		"manifest":      guid1.String(),
		VolumeIdListTag: volumeID,
	}

	volume, _ := provider.VolumeGet(ctx, volumeID, "")
	snapshot1, _ := provider.SnapshotCreate(ctx, *volume, map[string]string{"manifest": guid1.String(), DescriptionTag: guid2.String()})
	snapshot2, _ := provider.SnapshotCreate(ctx, *volume, tags)

	foundSnapshotsByID, _ := provider.SnapshotsList(ctx, tags)
	foundAllSnapshots, _ := provider.SnapshotsList(ctx, map[string]string{"manifest": guid1.String()})

	c.Assert(len(foundSnapshotsByID), check.Equals, 1)
	c.Assert(len(foundAllSnapshots), check.Equals, 2)
	c.Assert(snapshot2.ID, check.Equals, foundSnapshotsByID[0].ID)

	err := provider.SnapshotDelete(ctx, snapshot2)
	c.Assert(err, check.IsNil)
	err = provider.SnapshotDelete(ctx, snapshot1)
	c.Assert(err, check.IsNil)

	foundAllSnapshots, _ = provider.SnapshotsList(ctx, map[string]string{"manifest": guid1.String()})
	c.Assert(len(foundAllSnapshots), check.Equals, 0)
}
