// Copyright 2023 The Kanister Authors.
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

package blockstorage

import (
	. "gopkg.in/check.v1"
)

type HelperSuite struct{}

var _ = Suite(&HelperSuite{})

func (s *HelperSuite) SetUpSuite(c *C) {
}

func (h *HelperSuite) TestStringSlice(c *C) {
	source := []string{"test1", "test2"}
	target := StringSlice(&source)
	c.Assert(target[0], Equals, source[0])
	c.Assert(target[1], Equals, source[1])
}

func (s *HelperSuite) TestSliceStringPtr(c *C) {
	source := []string{"test1", "test2"}
	res := SliceStringPtr(source)
	for i, elePtr := range res {
		var target = *elePtr
		c.Assert(target, Equals, source[i])
	}
}

func (s *HelperSuite) TestIntFromPtr(c *C) {
	source := 1
	target := Int(&source)
	c.Assert(target, Equals, source)
}

func (s *HelperSuite) TestIntToPtr(c *C) {
	source := 1
	target := IntPtr(source)
	c.Assert(*target, Equals, source)
}

func (s *HelperSuite) TestStringToPtr(c *C) {
	source := "test"
	target := StringPtr(source)
	c.Assert(*target, Equals, source)
}
