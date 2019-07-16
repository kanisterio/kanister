package awsefs

import (
	"context"
	"testing"

	"github.com/kanisterio/kanister/pkg/blockstorage"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type AWSEFSTestSuite struct {
	provider blockstorage.Provider
}

func (s *AWSEFSTestSuite) SetUpSuite(c *C) {
	s.provider = NewEFSProvider()
}
func (s *AWSEFSTestSuite) TestVolumeCreateGetDelete(c *C) {
	c.Skip("Implementation is not ready")
	ctx := context.Background()
	az := "us-west-2a"
	tags := make(map[string]string)
	vol := blockstorage.Volume{
		Az:         az,
		Type:       blockstorage.TypeEFS,
		Attributes: tags,
	}

	vol1, err := s.provider.VolumeCreate(ctx, vol)
	c.Assert(err, IsNil)
	c.Assert(vol1.ID, Equals, vol.ID)

	vol2, err := s.provider.VolumeGet(ctx, vol1.ID, vol1.Az)
	c.Assert(err, IsNil)
	c.Assert(vol2.ID, Equals, vol1.ID)

	vols1, err := s.provider.VolumesList(ctx, tags, vol2.Az)
	c.Assert(err, IsNil)
	c.Assert(vols1, HasLen, 1)

	err = s.provider.VolumeDelete(ctx, vol2)
	c.Assert(err, IsNil)

	// Deleting second time must not fail
	err = s.provider.VolumeDelete(ctx, vol2)
	c.Assert(err, IsNil)

	// Get for deleted volume must return error
	_, err = s.provider.VolumeGet(ctx, vol2.ID, vol2.Az)
	c.Assert(err, NotNil)

	vols2, err := s.provider.VolumesList(ctx, tags, vol2.Az)
	c.Assert(err, IsNil)
	c.Assert(vols2, HasLen, 0)
}
