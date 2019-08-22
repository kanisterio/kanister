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

// +build !unit

package ibm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	. "gopkg.in/check.v1"
)

const (
	testTomlPath  = "testdata/correct"
	testBogusPath = "testdata/incorrect"
	workAroundEnv = "IBM_STORE_TOML"
	IBMApiKeyEnv  = "IBM_API_KEY"
)

//These are not executed as part of Pipeline, but usefull for development
// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type ClientSuite struct {
	apiKey string
}

var _ = Suite(&ClientSuite{})

func (s *ClientSuite) SetUpSuite(c *C) {
	if os.Getenv(IBMApiKeyEnv) == "" {
		c.Skip(IBMApiKeyEnv + " envionment variable not set")
	}
	s.apiKey = os.Getenv(IBMApiKeyEnv)
}

func (s *ClientSuite) TearDownSuite(c *C) {
	os.Setenv(IBMApiKeyEnv, s.apiKey)
	os.Unsetenv(LibDefCfgEnv)
}

func (s *ClientSuite) TestAPIClient(c *C) {
	var apiKey string
	if apiK, ok := os.LookupEnv(IBMApiKeyEnv); ok {
		apiKey = apiK
	} else {
		c.Skip(fmt.Sprintf("Could not find env var %s with API key", IBMApiKeyEnv))
	}
	ibmCli, err := newClient(context.Background(), map[string]string{APIKeyArgName: apiKey})
	c.Assert(err, IsNil)
	c.Assert(ibmCli, NotNil)
	c.Assert(ibmCli.Service, NotNil)
	defer ibmCli.Service.Close()
	c.Assert(*ibmCli, FitsTypeOf, client{})
	_, err = ibmCli.Service.ListSnapshots()
	c.Assert(err, IsNil)
}

func (s *ClientSuite) TestIBMClientSoftlayerFile(c *C) {
	var apiKey string
	if apiK, ok := os.LookupEnv(IBMApiKeyEnv); ok {
		apiKey = apiK
	} else {
		c.Skip(fmt.Sprintf("Could not find env var %s with API key", IBMApiKeyEnv))
	}
	ibmCli, err := newClient(context.Background(), map[string]string{APIKeyArgName: apiKey, SoftlayerFileAttName: "true"})
	defer ibmCli.Service.Close()
	c.Assert(err, IsNil)
	c.Assert(ibmCli.Service, NotNil)
	c.Assert(*ibmCli, FitsTypeOf, client{})
	c.Assert(ibmCli.SLCfg.SoftlayerBlockEnabled, Equals, false)
	c.Assert(ibmCli.SLCfg.SoftlayerFileEnabled, Equals, true)
	_, err = ibmCli.Service.ListSnapshots()
	c.Assert(err, IsNil)
}

func (s *ClientSuite) TestDefaultLibConfig(c *C) {
	if tomlPath, ok := os.LookupEnv(workAroundEnv); ok {
		err := os.Setenv(LibDefCfgEnv, filepath.Dir(tomlPath))
		c.Assert(err, IsNil)
		defer os.Unsetenv(LibDefCfgEnv)
	} else {
		c.Skip(workAroundEnv + " TOML path is not present")
	}
	apiKey := os.Getenv(IBMApiKeyEnv)
	err := os.Unsetenv(IBMApiKeyEnv)
	c.Assert(err, IsNil)
	defer os.Setenv(IBMApiKeyEnv, apiKey)
	ibmCli, err := newClient(context.Background(), make(map[string]string))
	c.Assert(err, IsNil)
	c.Assert(ibmCli, NotNil)
	c.Assert(ibmCli.Service, NotNil)
	defer ibmCli.Service.Close()
	c.Assert(*ibmCli, FitsTypeOf, client{})
}

func (s *ClientSuite) TestErrorsCases(c *C) {
	// Testing for bad secret or not present kubectl
	ibmCli, err := newClient(context.Background(), map[string]string{CfgSecretNameArgName: "somename"})
	c.Assert(err, NotNil)
	c.Assert(ibmCli, IsNil)
	err = os.Setenv(LibDefCfgEnv, "someboguspath")
	c.Assert(err, IsNil)
	ibmCli, err = newClient(context.Background(), make(map[string]string))
	c.Assert(err, NotNil)
	c.Assert(ibmCli, IsNil)
	err = os.Setenv(LibDefCfgEnv, testBogusPath)
	c.Assert(err, IsNil)
	ibmCli, err = newClient(context.Background(), make(map[string]string))
	c.Assert(err, NotNil)
	c.Assert(ibmCli, IsNil)
}
