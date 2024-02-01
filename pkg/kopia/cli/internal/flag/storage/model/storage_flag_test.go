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
