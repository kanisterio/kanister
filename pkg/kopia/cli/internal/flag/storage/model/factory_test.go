// Copyright 2024 The Kanister Authors.
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

package model

import (
	"testing"

	"gopkg.in/check.v1"

	"github.com/kanisterio/safecli"

	rs "github.com/kanisterio/kanister/pkg/secrets/repositoryserver"
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
