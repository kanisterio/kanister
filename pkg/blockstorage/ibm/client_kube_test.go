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
	"io/ioutil"
	"os"

	. "gopkg.in/check.v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/kube"
)

const (
	testSecretName    = "unitetestsecret"
	testOldSecretName = "oldibmsecret"
	oldTestTomlPath   = "./testdata/correct/libconfig_old.toml"
)

type KubeTestIBMClient struct {
	k8sSec *v1.Secret
	k8scli kubernetes.Interface
}

var _ = Suite(&KubeTestIBMClient{})

func (s *KubeTestIBMClient) SetUpSuite(c *C) {
	var secData []byte
	var err error
	if tomlPath, ok := os.LookupEnv(workAroundEnv); ok {
		secData, err = ioutil.ReadFile(tomlPath)
		c.Assert(err, IsNil)
	} else {
		c.Skip(workAroundEnv + " TOML path is not present")
	}

	secretData := make(map[string][]byte)
	secretData[IBMK8sSecretData] = secData

	s.k8scli, err = kube.NewClient()
	c.Assert(err, IsNil)
	k8sSec := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: testSecretName,
		},
		Type: v1.SecretTypeOpaque,
		Data: secretData,
	}
	s.k8sSec, err = s.k8scli.CoreV1().Secrets(IBMK8sSecretNS).Create(&k8sSec)
	c.Assert(err, IsNil)
}

func (s KubeTestIBMClient) TearDownSuite(c *C) {
	if _, ok := os.LookupEnv(workAroundEnv); !ok {
		c.Skip(workAroundEnv + "TOML path is not present")
	}
	err := s.k8scli.CoreV1().Secrets(s.k8sSec.Namespace).Delete(s.k8sSec.Name, &metav1.DeleteOptions{})
	c.Assert(err, IsNil)
}

func (s KubeTestIBMClient) TestIBMSecret(c *C) {
	apiKey := os.Getenv(IBMApiKeyEnv)
	err := os.Unsetenv(IBMApiKeyEnv)
	c.Assert(err, IsNil)
	defer os.Setenv(IBMApiKeyEnv, apiKey)
	ibmCli, err := newClient(context.Background(), map[string]string{CfgSecretNameArgName: testSecretName})
	c.Assert(err, IsNil)
	c.Assert(ibmCli, NotNil)
	c.Assert(ibmCli.Service, NotNil)
	defer ibmCli.Service.Close()
	c.Assert(*ibmCli, FitsTypeOf, client{})
	_, err = ibmCli.Service.ListSnapshots()
	c.Assert(err, IsNil)
}

func (s KubeTestIBMClient) TestIBMOldSecret(c *C) {
	apiKey := os.Getenv(IBMApiKeyEnv)
	err := os.Unsetenv(IBMApiKeyEnv)
	c.Assert(err, IsNil)
	defer os.Setenv(IBMApiKeyEnv, apiKey)
	secData, err := ioutil.ReadFile(oldTestTomlPath)
	c.Assert(err, IsNil)
	secretData := make(map[string][]byte)
	secretData[IBMK8sSecretData] = secData

	s.k8scli, err = kube.NewClient()
	c.Assert(err, IsNil)
	k8sSec := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: testOldSecretName,
		},
		Type: v1.SecretTypeOpaque,
		Data: secretData,
	}
	s.k8sSec, err = s.k8scli.CoreV1().Secrets(IBMK8sSecretNS).Create(&k8sSec)
	defer func() {
		_ = s.k8scli.CoreV1().Secrets(IBMK8sSecretNS).Delete(testOldSecretName, &metav1.DeleteOptions{})
	}()
	c.Assert(err, IsNil)
	slAPIKey, ok := os.LookupEnv(IBMSLApiKeyEnv)
	c.Check(slAPIKey, NotNil)
	c.Check(ok, Equals, true)
	slAPIUsername, ok := os.LookupEnv(IBMSLApiUsernameEnv)
	c.Check(slAPIUsername, NotNil)
	c.Check(ok, Equals, true)
	ibmCli, err := newClient(context.Background(), map[string]string{CfgSecretNameArgName: testOldSecretName, SLAPIKeyArgName: slAPIKey, SLAPIUsernameArgName: slAPIUsername})
	c.Assert(err, IsNil)
	c.Assert(ibmCli, NotNil)
	c.Assert(ibmCli.Service, NotNil)
	defer ibmCli.Service.Close()
	defer func() {
		_ = s.k8scli.CoreV1().Secrets(IBMK8sSecretNS).Delete(testOldSecretName, &metav1.DeleteOptions{})
	}()
	c.Assert(*ibmCli, FitsTypeOf, client{})
	_, err = ibmCli.Service.ListSnapshots()
	c.Assert(err, IsNil)
}

func (s *KubeTestIBMClient) TestSecretWSLApiKey(c *C) {
	testSlKey := "TestSlKey"
	testSlUserName := "TestUserName"
	ibmCfg, err := getDefIBMStoreSecret(context.Background(), map[string]string{CfgSecretNameArgName: testSecretName, SLAPIKeyArgName: testSlKey, SLAPIUsernameArgName: testSlUserName})
	c.Assert(err, IsNil)
	c.Assert(ibmCfg, NotNil)
	c.Assert(ibmCfg.Softlayer.SoftlayerAPIKey, Equals, testSlKey)
	c.Assert(ibmCfg.Softlayer.SoftlayerUsername, Equals, testSlUserName)
}
