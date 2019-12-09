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

// +build !unit

package ibm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	ibmcfg "github.com/IBM/ibmcloud-storage-volume-lib/config"

	. "gopkg.in/check.v1"
)

const (
	testBogusPath       = "testdata/incorrect"
	workAroundEnv       = "IBM_STORE_TOML"
	IBMApiKeyEnv        = "IBM_API_KEY"
	IBMSLApiKeyEnv      = "IBM_SL_API_KEY"
	IBMSLApiUsernameEnv = "IBM_SL_API_USERNAME"
)

//These are not executed as part of Pipeline, but usefull for development
// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type ClientSuite struct {
	apiKey        string
	slAPIKey      string
	slAPIUsername string
}

var _ = Suite(&ClientSuite{})

func (s *ClientSuite) SetUpSuite(c *C) {
	var ok bool
	if s.slAPIKey, ok = os.LookupEnv(IBMSLApiKeyEnv); ok {
		if s.slAPIUsername, ok = os.LookupEnv(IBMSLApiUsernameEnv); ok {
			return
		}
	}
	if s.apiKey, ok = os.LookupEnv(IBMApiKeyEnv); ok {
		return
	}
	c.Skip(fmt.Sprintf("One of  %s, %s and %s environment variable is not set", IBMApiKeyEnv, IBMSLApiKeyEnv, IBMSLApiUsernameEnv))
}

func (s *ClientSuite) TearDownSuite(c *C) {
	os.Setenv(IBMApiKeyEnv, s.apiKey)
	os.Unsetenv(LibDefCfgEnv)
}

func (s *ClientSuite) TestAPIClient(c *C) {
	if tomlPath, ok := os.LookupEnv(workAroundEnv); ok {
		err := os.Setenv(LibDefCfgEnv, filepath.Dir(tomlPath))
		c.Assert(err, IsNil)
		defer os.Unsetenv(LibDefCfgEnv)
	} else {
		c.Skip(workAroundEnv + " TOML path is not present")
	}
	args := s.getCredsMap(c)
	ibmCli, err := newClient(context.Background(), args)
	c.Assert(err, IsNil)
	c.Assert(ibmCli, NotNil)
	c.Assert(ibmCli.Service, NotNil)
	defer ibmCli.Service.Close()
	c.Assert(*ibmCli, FitsTypeOf, client{})
	_, err = ibmCli.Service.ListSnapshots()
	c.Assert(err, IsNil)
}

func (s *ClientSuite) TestIBMClientSoftlayerFile(c *C) {
	if tomlPath, ok := os.LookupEnv(workAroundEnv); ok {
		err := os.Setenv(LibDefCfgEnv, filepath.Dir(tomlPath))
		c.Assert(err, IsNil)
		defer os.Unsetenv(LibDefCfgEnv)
	} else {
		c.Skip(workAroundEnv + " TOML path is not present")
	}
	args := s.getCredsMap(c)
	args[SoftlayerFileAttName] = "true"
	ibmCli, err := newClient(context.Background(), args)
	c.Assert(err, IsNil)
	defer ibmCli.Service.Close()
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
	defer os.Unsetenv(LibDefCfgEnv)
	c.Assert(err, IsNil)
	ibmCli, err = newClient(context.Background(), make(map[string]string))
	c.Assert(err, NotNil)
	c.Assert(ibmCli, IsNil)

}

func (s *ClientSuite) getCredsMap(c *C) map[string]string {
	if s.slAPIKey != "" {
		return map[string]string{SLAPIKeyArgName: s.slAPIKey, SLAPIUsernameArgName: s.slAPIUsername}
	}
	if s.apiKey != "" {
		return map[string]string{APIKeyArgName: s.apiKey}
	}
	c.Skip(fmt.Sprintf("Neither of  %s, %s  environment variables set", IBMApiKeyEnv, IBMSLApiKeyEnv))
	return map[string]string{}
}

func (s *ClientSuite) TestPanic(c *C) {
	for _, f := range []func() (*client, error){
		func() (*client, error) {
			panic("TEST")
		},
		func() (*client, error) {
			var cfg *client
			cfg.SLCfg = ibmcfg.SoftlayerConfig{}
			return nil, nil
		},
		func() (*client, error) {
			var x []int
			x[0]++
			return nil, nil
		},
	} {
		cfg, err := handleClientPanic(f)
		c.Assert(err, NotNil)
		c.Assert(cfg, IsNil)
	}
}
