package model

import (
	"testing"

	"github.com/kanisterio/kanister/pkg/safecli"
	rs "github.com/kanisterio/kanister/pkg/secrets/repositoryserver"
	"gopkg.in/check.v1"
)

func TestStorageFlag(t *testing.T) { check.TestingT(t) }

type StorageFlagSuite struct{}

var _ = check.Suite(&StorageFlagSuite{})

func (s *StorageFlagSuite) TestGetLogger(c *check.C) {
	sf := StorageFlag{}
	c.Check(sf.GetLogger(), check.NotNil)
	sf.Logger = nil
	c.Check(sf.GetLogger(), check.NotNil)
}

func (s *StorageFlagSuite) TestApplyNoFactory(c *check.C) {
	sf := StorageFlag{}
	err := sf.Apply(nil)
	c.Check(err, check.Equals, ErrInvalidFactor)
}

func (s *StorageFlagSuite) TestApply(c *check.C) {
	sf := StorageFlag{
		Location: Location{
			rs.TypeKey: []byte("blue"),
		},
		Factory: mockFactory(),
	}
	b := safecli.NewBuilder()
	err := sf.Apply(b)
	c.Check(err, check.IsNil)
	c.Check(b.Build(), check.DeepEquals, []string{"blue"})
}

func (s *StorageFlagSuite) TestApplyUnknowType(c *check.C) {
	sf := StorageFlag{
		Location: Location{
			rs.TypeKey: []byte("unknow"),
		},
		Factory: mockFactory(),
	}
	b := safecli.NewBuilder()
	err := sf.Apply(b)
	c.Check(err, check.ErrorMatches, ".*failed to apply storage args.*")
}
