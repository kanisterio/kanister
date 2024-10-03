// Copyright 2020 The Kanister Authors.
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

package virtualfs

import (
	"testing"

	"gopkg.in/check.v1"
)

func Test(t *testing.T) { check.TestingT(t) }

type VirtualFSSuite struct{}

var _ = check.Suite(&VirtualFSSuite{})

func (s *VirtualFSSuite) TestNewDirectory(c *check.C) {
	for _, tc := range []struct {
		caseName string
		rootName string
		checker  check.Checker
	}{
		{
			caseName: "Root Directory success",
			rootName: "root",
			checker:  check.IsNil,
		},
		{
			caseName: "Root directory with `/`",
			rootName: "/root",
			checker:  check.NotNil,
		},
	} {
		r, err := NewDirectory(tc.rootName)
		c.Check(err, tc.checker, check.Commentf("Case %s failed", tc.caseName))
		if err == nil {
			c.Check(r.Name(), check.Equals, tc.rootName, check.Commentf("Case %s failed", tc.caseName))
		}
	}
}
