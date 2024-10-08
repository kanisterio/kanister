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

package mockblockstorage

import (
	"testing"

	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/blockstorage"
)

func Test(t *testing.T) { check.TestingT(t) }

type MockSuite struct{}

var _ = check.Suite(&MockSuite{})

func (s *MockSuite) TestMockStorage(c *check.C) {
	mock, err := Get(blockstorage.TypeEBS)
	c.Assert(err, check.IsNil)
	c.Assert(mock.Type(), check.Equals, blockstorage.TypeEBS)
}
