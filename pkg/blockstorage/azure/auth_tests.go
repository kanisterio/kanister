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
	"github.com/kanisterio/kanister/pkg/blockstorage"
	. "gopkg.in/check.v1"
)

type AuthSuite struct{}

var _ = Suite(&AuthSuite{})

func (s *AuthSuite) SetUpSuite(c *C) {
}

func (s *AuthSuite) TestIsClientCredsvailable(c *C) {
	// success
	config := map[string]string{
		blockstorage.AzureTenantID:    "some-tenant-id",
		blockstorage.AzureCientID:     "some-client-id",
		blockstorage.AzureClentSecret: "someclient-secret",
	}
	c.Assert(isClientCredsAvailable(config), Equals, true)

	// remove tenantID
	delete(config, blockstorage.AzureTenantID)
	c.Assert(isClientCredsAvailable(config), Equals, false)

	// remove client secret, only client ID left
	delete(config, blockstorage.AzureClentSecret)
	c.Assert(isClientCredsAvailable(config), Equals, false)
}

func (s *AuthSuite) TestIsMSICredsAvailable(c *C) {
	// success
	config := map[string]string{
		blockstorage.AzureTenantID:    "some-tenant-id",
		blockstorage.AzureCientID:     "some-client-id",
		blockstorage.AzureClentSecret: "someclient-secret",
	}
	c.Assert(isMSICredsAvailable(config), Equals, false)

	// remove tenantID
	delete(config, blockstorage.AzureTenantID)
	c.Assert(isMSICredsAvailable(config), Equals, false)

	// remove client secret, only client ID left
	delete(config, blockstorage.AzureClentSecret)
	c.Assert(isMSICredsAvailable(config), Equals, true)
}

func (s *AuthSuite) TestNewAzureAutheticator(c *C) {
	// successful with client secret creds
	config := map[string]string{
		blockstorage.AzureTenantID:    "some-tenant-id",
		blockstorage.AzureCientID:     "some-client-id",
		blockstorage.AzureClentSecret: "some-client-secret",
	}
	authenticator, err := NewAzureAuthenticator(config)
	c.Assert(err, IsNil)
	_, ok := authenticator.(*ClientSecretAuthenticator)
	c.Assert(ok, Equals, true)

	// successful with msi creds
	config = map[string]string{
		blockstorage.AzureCientID: "some-client-id",
	}
	authenticator, err = NewAzureAuthenticator(config)
	c.Assert(err, IsNil)
	_, ok = authenticator.(*MsiAuthenticator)
	c.Assert(ok, Equals, true)

	// unsuccessful with an undefined combo of creds
	config = map[string]string{
		blockstorage.AzureCientID:     "some-client-id",
		blockstorage.AzureClentSecret: "some-client-secret",
	}
	authenticator, err = NewAzureAuthenticator(config)
	c.Assert(err, NotNil)
	c.Assert(authenticator, IsNil)
}
