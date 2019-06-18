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
	testSecretName = "unitetestsecret"
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
	_, err = ibmCli.Service.SnapshotsList()
	c.Assert(err, IsNil)
}
