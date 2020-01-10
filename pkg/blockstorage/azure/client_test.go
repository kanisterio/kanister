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

package azure

import (
	"context"
	"testing"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type ClientSuite struct{}

var _ = Suite(&ClientSuite{})

func (s *ClientSuite) SetUpSuite(c *C) {
}

func (s *ClientSuite) TestClient(c *C) {
	c.Skip("Until Azure will be fully integrated into build.sh")
	azCli, err := NewClient(context.Background())
	c.Assert(err, IsNil)

	c.Assert(azCli.SubscriptionID, NotNil)
	c.Assert(azCli.Authorizer, NotNil)
	c.Assert(azCli.DisksClient, NotNil)
	c.Assert(azCli.SnapshotsClient, NotNil)
	_, err = azCli.DisksClient.List(context.Background())
	c.Assert(err, IsNil)
}
