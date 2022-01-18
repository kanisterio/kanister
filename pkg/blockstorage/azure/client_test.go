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
	"fmt"
	"strings"
	"testing"

	"github.com/kanisterio/kanister/pkg/blockstorage"
	envconfig "github.com/kanisterio/kanister/pkg/config"
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
	config := make(map[string]string)
	config[blockstorage.AzureSubscriptionID] = envconfig.GetEnvOrSkip(c, blockstorage.AzureSubscriptionID)
	config[blockstorage.AzureTenantID] = envconfig.GetEnvOrSkip(c, blockstorage.AzureTenantID)
	config[blockstorage.AzureCientID] = envconfig.GetEnvOrSkip(c, blockstorage.AzureCientID)
	config[blockstorage.AzureClentSecret] = envconfig.GetEnvOrSkip(c, blockstorage.AzureClentSecret)
	config[blockstorage.AzureResurceGroup] = envconfig.GetEnvOrSkip(c, blockstorage.AzureResurceGroup)
	config[blockstorage.AzureCloudEnviornmentID] = envconfig.GetEnvOrSkip(c, blockstorage.AzureCloudEnviornmentID)
	azCli, err := NewClient(context.Background(), config)
	c.Assert(err, IsNil)

	c.Assert(azCli.SubscriptionID, NotNil)
	c.Assert(azCli.Authorizer, NotNil)
	c.Assert(azCli.DisksClient, NotNil)
	c.Assert(azCli.SnapshotsClient, NotNil)
	_, err = azCli.DisksClient.List(context.Background())
	c.Assert(err, IsNil)
}

func (s ClientSuite) TestGetRegions(c *C) {
	ctx := context.Background()
	config := map[string]string{}
	config[blockstorage.AzureSubscriptionID] = envconfig.GetEnvOrSkip(c, blockstorage.AzureSubscriptionID)
	config[blockstorage.AzureTenantID] = envconfig.GetEnvOrSkip(c, blockstorage.AzureTenantID)
	config[blockstorage.AzureCientID] = envconfig.GetEnvOrSkip(c, blockstorage.AzureCientID)
	config[blockstorage.AzureClentSecret] = envconfig.GetEnvOrSkip(c, blockstorage.AzureClentSecret)
	config[blockstorage.AzureResurceGroup] = envconfig.GetEnvOrSkip(c, blockstorage.AzureResurceGroup)
	// config[blockstorage.AzureCloudEnviornmentID] = envconfig.GetEnvOrSkip(c, blockstorage.AzureCloudEnviornmentID)

	bsp, err := NewProvider(ctx, config)
	c.Assert(err, IsNil)
	ads := bsp.(*AdStorage)

	// get zones with other region
	zones, err := ads.FromRegion(ctx, "eastus")
	fmt.Println(zones)
	c.Assert(err, IsNil)
	for _, zone := range zones {
		c.Assert(strings.Contains(zone, "eastus"), Equals, true)
	}

	regions, err := ads.GetRegions(ctx)
	c.Assert(err, IsNil)
	c.Assert(regions, NotNil)
}
