// Copyright 2019 Kasten Inc.
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

package awsebs

import (
	"context"
	"testing"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type AWSEBSSuite struct{}

var _ = Suite(&AWSEBSSuite{})

func (s AWSEBSSuite) TestQueryRegionToZones(c *C) {
	c.Skip("Only works on AWS")
	ctx := context.Background()
	region := "us-east-1"
	zs, err := queryRegionToZones(ctx, region)
	c.Assert(err, IsNil)
	c.Assert(zs, DeepEquals, []string{"us-east-1a", "us-east-1b", "us-east-1c", "us-east-1d", "us-east-1e", "us-east-1f"})
}
