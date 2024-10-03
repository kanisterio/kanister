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

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/blockstorage"
	envconfig "github.com/kanisterio/kanister/pkg/config"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { check.TestingT(t) }

type ClientSuite struct{}

var _ = check.Suite(&ClientSuite{})

func (s *ClientSuite) SetUpSuite(c *check.C) {
}

func (s *ClientSuite) TestClient(c *check.C) {
	c.Skip("Until Azure will be fully integrated into build.sh")
	config := make(map[string]string)
	config[blockstorage.AzureSubscriptionID] = envconfig.GetEnvOrSkip(c, blockstorage.AzureSubscriptionID)
	config[blockstorage.AzureTenantID] = envconfig.GetEnvOrSkip(c, blockstorage.AzureTenantID)
	config[blockstorage.AzureClientID] = envconfig.GetEnvOrSkip(c, blockstorage.AzureClientID)
	config[blockstorage.AzureClientSecret] = envconfig.GetEnvOrSkip(c, blockstorage.AzureClientSecret)
	config[blockstorage.AzureResurceGroup] = envconfig.GetEnvOrSkip(c, blockstorage.AzureResurceGroup)
	config[blockstorage.AzureCloudEnvironmentID] = envconfig.GetEnvOrSkip(c, blockstorage.AzureCloudEnvironmentID)
	azCli, err := NewClient(context.Background(), config)
	c.Assert(err, check.IsNil)
	c.Assert(azCli.Cred, check.NotNil)
	c.Assert(azCli.SubscriptionID, check.NotNil)
	c.Assert(azCli.DisksClient, check.NotNil)
	c.Assert(azCli.SnapshotsClient, check.NotNil)
	c.Assert(azCli.DisksClient.NewListPager(nil), check.NotNil)
	c.Assert(azCli.SKUsClient, check.NotNil)
	c.Assert(azCli.SubscriptionsClient, check.NotNil)
	c.Assert(err, check.IsNil)
}

func (s ClientSuite) TestGetRegions(c *check.C) {
	ctx := context.Background()
	config := map[string]string{}
	config[blockstorage.AzureSubscriptionID] = envconfig.GetEnvOrSkip(c, blockstorage.AzureSubscriptionID)
	config[blockstorage.AzureTenantID] = envconfig.GetEnvOrSkip(c, blockstorage.AzureTenantID)
	config[blockstorage.AzureClientID] = envconfig.GetEnvOrSkip(c, blockstorage.AzureClientID)
	config[blockstorage.AzureClientSecret] = envconfig.GetEnvOrSkip(c, blockstorage.AzureClientSecret)
	config[blockstorage.AzureResurceGroup] = envconfig.GetEnvOrSkip(c, blockstorage.AzureResurceGroup)
	// config[blockstorage.AzureCloudEnviornmentID] = envconfig.GetEnvOrSkip(c, blockstorage.AzureCloudEnviornmentID)

	bsp, err := NewProvider(ctx, config)
	c.Assert(err, check.IsNil)
	ads := bsp.(*AdStorage)

	// get zones with other region
	zones, err := ads.FromRegion(ctx, "eastus")
	fmt.Println(zones)
	c.Assert(err, check.IsNil)
	for _, zone := range zones {
		c.Assert(strings.Contains(zone, "eastus"), check.Equals, true)
		c.Assert(strings.Contains(zone, "westus"), check.Equals, false)
	}

	regions, err := ads.GetRegions(ctx)
	c.Assert(err, check.IsNil)
	c.Assert(regions, check.NotNil)
}

func (s *ClientSuite) TestGetCredConfig(c *check.C) {
	for _, tc := range []struct {
		name       string
		env        Environment
		config     map[string]string
		errChecker check.Checker
		expCCC     ClientCredentialsConfig
	}{
		{
			name: "Test with all attributes in configuration",
			env:  PublicCloud,
			config: map[string]string{
				blockstorage.AzureTenantID:            "atid",
				blockstorage.AzureClientID:            "acid",
				blockstorage.AzureClientSecret:        "acs",
				blockstorage.AzureActiveDirEndpoint:   "aade",
				blockstorage.AzureActiveDirResourceID: "aadrid",
			},
			expCCC: ClientCredentialsConfig{
				ClientID:     "acid",
				ClientSecret: "acs",
				TenantID:     "atid",
				Resource:     "aadrid",
				AADEndpoint:  "aade",
			},
			errChecker: check.IsNil,
		},
		{
			name: "Test with client credential in configuration",
			env:  PublicCloud,
			config: map[string]string{
				blockstorage.AzureTenantID:     "atid",
				blockstorage.AzureClientID:     "acid",
				blockstorage.AzureClientSecret: "acs",
			},
			expCCC: ClientCredentialsConfig{
				ClientID:     "acid",
				ClientSecret: "acs",
				TenantID:     "atid",
				Resource:     cloud.AzurePublic.Services[cloud.ResourceManager].Endpoint,
				AADEndpoint:  cloud.AzurePublic.ActiveDirectoryAuthorityHost,
			},
			errChecker: check.IsNil,
		},
		{
			name: "Test without AD in configuration",
			env:  USGovernmentCloud,
			config: map[string]string{
				blockstorage.AzureTenantID:            "atid",
				blockstorage.AzureClientID:            "acid",
				blockstorage.AzureClientSecret:        "acs",
				blockstorage.AzureActiveDirEndpoint:   "",
				blockstorage.AzureActiveDirResourceID: "",
			},
			expCCC: ClientCredentialsConfig{
				ClientID:     "acid",
				ClientSecret: "acs",
				TenantID:     "atid",
				Resource:     cloud.AzureGovernment.Services[cloud.ResourceManager].Endpoint,
				AADEndpoint:  cloud.AzureGovernment.ActiveDirectoryAuthorityHost,
			},
			errChecker: check.IsNil,
		},
		{
			name: "Test with tenantid and clientid in configuration",
			env:  USGovernmentCloud,
			config: map[string]string{
				blockstorage.AzureTenantID: "atid",
				blockstorage.AzureClientID: "acid",
			},
			errChecker: check.NotNil,
		},
		{
			name: "Test with tenantid in configuration",
			env:  USGovernmentCloud,
			config: map[string]string{
				blockstorage.AzureTenantID: "atid",
			},
			errChecker: check.NotNil,
		},
		{
			name:       "Test with nil configuration",
			env:        USGovernmentCloud,
			config:     map[string]string{},
			errChecker: check.NotNil,
		},
	} {
		ccc, err := getCredConfig(tc.env, tc.config)
		c.Assert(err, tc.errChecker)
		if err == nil {
			c.Assert(ccc.ClientID, check.Equals, tc.expCCC.ClientID)
			c.Assert(ccc.ClientSecret, check.Equals, tc.expCCC.ClientSecret)
			c.Assert(ccc.TenantID, check.Equals, tc.expCCC.TenantID)
			c.Assert(ccc.Resource, check.Equals, tc.expCCC.Resource)
			c.Assert(ccc.AADEndpoint, check.Equals, tc.expCCC.AADEndpoint)
		}
	}
}
