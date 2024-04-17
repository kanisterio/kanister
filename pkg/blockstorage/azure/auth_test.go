// Copyright 2022 The Kanister Authors.
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
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	. "gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/blockstorage"
	"github.com/kanisterio/kanister/pkg/consts"
)

type AuthSuite struct{}

var _ = Suite(&AuthSuite{})

func (s *AuthSuite) SetUpSuite(c *C) {
}

func (s *AuthSuite) TestIsClientCredsvailable(c *C) {
	// success
	config := map[string]string{
		blockstorage.AzureTenantID:     "some-tenant-id",
		blockstorage.AzureClientID:     "some-client-id",
		blockstorage.AzureClientSecret: "someclient-secret",
	}
	c.Assert(isClientCredsAvailable(config), Equals, true)

	// remove tenantID
	delete(config, blockstorage.AzureTenantID)
	c.Assert(isClientCredsAvailable(config), Equals, false)

	// remove client secret, only client ID left
	delete(config, blockstorage.AzureClientSecret)
	c.Assert(isClientCredsAvailable(config), Equals, false)
}

func (s *AuthSuite) TestIsMSICredsAvailable(c *C) {
	// success
	config := map[string]string{
		blockstorage.AzureTenantID:     "some-tenant-id",
		blockstorage.AzureClientID:     "some-client-id",
		blockstorage.AzureClientSecret: "someclient-secret",
	}
	c.Assert(isMSICredsAvailable(config), Equals, false)

	// remove tenantID
	delete(config, blockstorage.AzureTenantID)
	c.Assert(isMSICredsAvailable(config), Equals, false)

	// remove client secret, only client ID left
	delete(config, blockstorage.AzureClientSecret)
	c.Assert(isMSICredsAvailable(config), Equals, true)

	// empty client ID - default msi id is implied
	config = map[string]string{
		blockstorage.AzureClientID: "",
	}
	c.Assert(isMSICredsAvailable(config), Equals, true)

	// empty creds
	config = map[string]string{}
	c.Assert(isMSICredsAvailable(config), Equals, false)
}

func (s *AuthSuite) TestNewAzureAuthenticator(c *C) {
	// successful with client secret creds
	config := map[string]string{
		blockstorage.AzureTenantID:     "some-tenant-id",
		blockstorage.AzureClientID:     "some-client-id",
		blockstorage.AzureClientSecret: "some-client-secret",
	}
	authenticator, err := NewAzureAuthenticator(config, nil)
	c.Assert(err, IsNil)
	_, ok := authenticator.(*ClientSecretAuthenticator)
	c.Assert(ok, Equals, true)

	// successful with msi creds
	config = map[string]string{
		blockstorage.AzureClientID: "some-client-id",
	}
	authenticator, err = NewAzureAuthenticator(config, nil)
	c.Assert(err, IsNil)
	_, ok = authenticator.(*MsiAuthenticator)
	c.Assert(ok, Equals, true)

	// successful with default msi creds
	config = map[string]string{
		blockstorage.AzureClientID: "",
	}
	authenticator, err = NewAzureAuthenticator(config, nil)
	c.Assert(err, IsNil)
	c.Assert(authenticator, NotNil)

	// unsuccessful with no creds
	config = map[string]string{}
	authenticator, err = NewAzureAuthenticator(config, nil)
	c.Assert(err, NotNil)
	c.Assert(authenticator, IsNil)

	// unsuccessful with an undefined combo of credss
	config = map[string]string{
		blockstorage.AzureClientSecret: "some-client-secret",
	}
	authenticator, err = NewAzureAuthenticator(config, nil)
	c.Assert(err, NotNil)
	c.Assert(authenticator, IsNil)

	// unsuccessful with an undefined combo of creds
	config = map[string]string{
		blockstorage.AzureClientID:     "some-client-id",
		blockstorage.AzureClientSecret: "some-client-secret",
	}
	authenticator, err = NewAzureAuthenticator(config, nil)
	c.Assert(err, NotNil)
	c.Assert(authenticator, IsNil)
}

func (s *AuthSuite) TestNewAzureAuthenticatorCloudConfig(c *C) {
	msiCfg := map[string]string{
		blockstorage.AzureClientID: "id",
	}
	secretCfg := map[string]string{
		blockstorage.AzureClientID:     "id",
		blockstorage.AzureClientSecret: "sec",
		blockstorage.AzureTenantID:     "tenant",
	}

	for ti, tc := range []struct {
		name                string
		cfg                 map[string]string
		cloudEnv            string
		expectedCloudConfig cloud.Configuration
	}{
		{
			name:                "China env runs on China cloud for MSI",
			cfg:                 msiCfg,
			cloudEnv:            consts.AzureChinaCloud,
			expectedCloudConfig: cloud.AzureChina,
		},
		{
			name:                "USGov env runs on USGov cloud for MSI",
			cfg:                 msiCfg,
			cloudEnv:            consts.AzureUSGovernmentCloud,
			expectedCloudConfig: cloud.AzureGovernment,
		},
		{
			name:                "Unset env runs on public cloud for MSI",
			cfg:                 msiCfg,
			expectedCloudConfig: cloud.AzurePublic,
		},
		{
			name:                "China env runs on China cloud for client secret",
			cfg:                 secretCfg,
			cloudEnv:            consts.AzureChinaCloud,
			expectedCloudConfig: cloud.AzureChina,
		},
		{
			name:                "USGov env runs on USGov cloud for client secret",
			cfg:                 secretCfg,
			cloudEnv:            consts.AzureUSGovernmentCloud,
			expectedCloudConfig: cloud.AzureGovernment,
		},
		{
			name:                "Unset env runs on public cloud for client secret",
			cfg:                 secretCfg,
			expectedCloudConfig: cloud.AzurePublic,
		},
	} {
		c.Logf("%d: %s", ti, tc.name)
		newCfg := make(map[string]string)
		for k, v := range tc.cfg {
			newCfg[k] = v
		}
		newCfg[blockstorage.AzureCloudEnvironmentID] = tc.cloudEnv

		azIdentity := &AzIdentityType{
			NewManagedIdentityCredential: func(opts *azidentity.ManagedIdentityCredentialOptions) (*azidentity.ManagedIdentityCredential, error) {
				c.Assert(opts.ClientOptions.Cloud.ActiveDirectoryAuthorityHost, Equals, tc.expectedCloudConfig.ActiveDirectoryAuthorityHost)
				return nil, nil
			},
			NewClientSecretCredential: func(tenantID, clientID, clientSecret string, opts *azidentity.ClientSecretCredentialOptions) (*azidentity.ClientSecretCredential, error) {
				c.Assert(opts.ClientOptions.Cloud.ActiveDirectoryAuthorityHost, Equals, tc.expectedCloudConfig.ActiveDirectoryAuthorityHost)
				return nil, nil
			},
		}

		auth, err := NewAzureAuthenticator(newCfg, azIdentity)
		c.Assert(err, IsNil)

		err = auth.Authenticate(newCfg)
		c.Assert(err, IsNil)
	}
}
