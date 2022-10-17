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

package gcepd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	. "gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/blockstorage"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type ClientSuite struct{}

var _ = Suite(&ClientSuite{})

func (s *ClientSuite) SetUpSuite(c *C) {}

func (s *ClientSuite) TestClient(c *C) {
	var zone string
	filename := s.GetEnvOrSkip(c, blockstorage.GoogleCloudCreds)
	b, err := os.ReadFile(filename)
	c.Assert(err, IsNil)
	gCli, err := NewClient(context.Background(), string(b))
	c.Assert(err, IsNil)
	c.Assert(gCli.Service, NotNil)
	c.Assert(*gCli, FitsTypeOf, Client{})
	// Get zone
	zone = s.GetEnvOrSkip(c, blockstorage.GoogleCloudZone)
	_, err = gCli.Service.Disks.List(gCli.ProjectID, zone).Do()
	c.Assert(err, IsNil)
}

func (s *ClientSuite) GetEnvOrSkip(c *C, varName string) string {
	v := os.Getenv(varName)
	// Ensure the variable is set
	if v == "" {
		c.Skip("Required environment variable " + varName + " is not set")
	}
	return v
}

func (s ClientSuite) TestGetRegions(c *C) {
	ctx := context.Background()
	config := map[string]string{}
	creds := s.GetEnvOrSkip(c, blockstorage.GoogleCloudCreds)

	// create provider with region
	config[blockstorage.GoogleCloudCreds] = creds
	bsp, err := NewProvider(config)
	c.Assert(err, IsNil)
	gpds := bsp.(*GpdStorage)

	// get zones with other region
	zones, err := gpds.FromRegion(ctx, "us-east1")
	fmt.Println(zones)
	c.Assert(err, IsNil)
	for _, zone := range zones {
		c.Assert(strings.Contains(zone, "us-east1"), Equals, true)
		c.Assert(strings.Contains(zone, "us-west1"), Equals, false)
	}

	regions, err := gpds.GetRegions(ctx)
	c.Assert(err, IsNil)
	c.Assert(regions, NotNil)
}
