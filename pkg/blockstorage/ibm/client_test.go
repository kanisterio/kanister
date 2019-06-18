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

type ClientSuite struct{}

var _ = Suite(&ClientSuite{})

func (s *ClientSuite) SetUpSuite(c *C) {
	if os.Getenv(IBMApiKeyEnv) == "" {
		c.Skip(IBMApiKeyEnv + " envionment variable not set")
	}
}

func (s *ClientSuite) TestClient(c *C) {
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
	_, err = ibmCli.Service.SnapshotsList()
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
