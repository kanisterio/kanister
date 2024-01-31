package model

import (
	"testing"

	rs "github.com/kanisterio/kanister/pkg/secrets/repositoryserver"

	"github.com/kanisterio/kanister/pkg/safecli"
	"gopkg.in/check.v1"
)

func TestFactory(t *testing.T) { check.TestingT(t) }

type FactorySuite struct{}

var (
	_         = check.Suite(&FactorySuite{})
	ltBlue    = rs.LocType("blue")
	ltRed     = rs.LocType("red")
	ltUnknown = rs.LocType("unknown")
)

type mockBuilder struct {
	cmd string
	err error
}

func (m *mockBuilder) New() StorageBuilder {
	return func(sf StorageFlag) (*safecli.Builder, error) {
		if m.err != nil {
			return nil, m.err
		}
		b := safecli.NewBuilder()
		b.AppendLoggable(m.cmd)
		return b, nil
	}
}

func mockFactory() StorageBuilderFactory {
	factory := make(BuildersFactory)

	blue := mockBuilder{cmd: "blue"}
	red := mockBuilder{cmd: "red"}

	factory[ltBlue] = blue.New()
	factory[ltRed] = red.New()
	return factory
}

func (s *FactorySuite) TestBuildersFactory(c *check.C) {
	factory := mockFactory()

	b, err := factory.Create(ltBlue)(StorageFlag{})
	c.Assert(err, check.IsNil)
	c.Check(b, check.NotNil)
	c.Check(b.Build(), check.DeepEquals, []string{"blue"})

	b, err = factory.Create(ltRed)(StorageFlag{})
	c.Assert(err, check.IsNil)
	c.Check(b, check.NotNil)
	c.Check(b.Build(), check.DeepEquals, []string{"red"})

	b, err = factory.Create(ltUnknown)(StorageFlag{})
	c.Assert(err, check.ErrorMatches, ".*unsupported location type: 'unknown'.*")
	c.Check(b, check.IsNil)
}
