package vmware

import (
	"context"

	"gopkg.in/check.v1"
	. "gopkg.in/check.v1"
)

type VmwareManSuite struct{}

var _ = check.Suite(&VmwareManSuite{})

func (s *VmwareManSuite) TestCreateAndListSnapshots(c *C) {
	c.Skip("manual testing")
	volumeID := "b12b3f3d-43e8-46ac-b5ba-70ee634139da"
	provider, _ := NewProvider(map[string]string{
		VSphereEndpointKey: "",
		VSphereUsernameKey: "",
		VSpherePasswordKey: "",
	})
	ftpProvider := provider.(*FcdProvider)

	ftpProvider.SetCategoryID("K10:0c66728a-dd0d-11ec-9939-ca6a7623d809")
	ctx := context.Background()

	tags := map[string]string{
		DescriptionTag: "manifest1",
		"manifest":     "manifest1",
		VolumeIdTag:    volumeID,
	}

	volume, _ := provider.VolumeGet(ctx, volumeID, "")
	snapshot1, _ := provider.SnapshotCreate(ctx, *volume, map[string]string{"manifest": "manifest1"})
	snapshot2, _ := provider.SnapshotCreate(ctx, *volume, tags)

	foundSnapshotsByID, _ := provider.SnapshotsList(ctx, tags)
	foundAllSnapshots, _ := provider.SnapshotsList(ctx, map[string]string{"manifest": "manifest1"})

	c.Assert(len(foundSnapshotsByID), Equals, 1)
	c.Assert(len(foundAllSnapshots), Equals, 2)
	c.Assert(snapshot2.ID, Equals, foundSnapshotsByID[0].ID)

	err := provider.SnapshotDelete(ctx, snapshot2)
	c.Assert(err, check.IsNil)
	err = provider.SnapshotDelete(ctx, snapshot1)
	c.Assert(err, check.IsNil)

	foundAllSnapshots, _ = provider.SnapshotsList(ctx, map[string]string{"manifest": "manifest1"})
	c.Assert(len(foundAllSnapshots), Equals, 0)
}
