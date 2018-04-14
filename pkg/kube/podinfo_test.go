package kube

import (
	. "gopkg.in/check.v1"
	"io/ioutil"
	"os"

	"k8s.io/client-go/kubernetes/fake"
)

type PodInfoSuite struct{}

var _ = Suite(&PodInfoSuite{})

const testPodNamespace = "test-pod-namespace"
const testPodName = "test-pod-name"
const testPodSA = "test-pod-sa"

func (s *PodInfoSuite) TestGetControllerNamespaceFromEnv(c *C) {
	os.Setenv(podNSEnvVar, testPodNamespace)
	ns, err := GetControllerNamespace()
	c.Assert(err, IsNil)
	c.Assert(ns, Equals, testPodNamespace)
	os.Unsetenv(podNSEnvVar)
}

func (s *PodInfoSuite) TestGetControllerNamespaceFromFile(c *C) {
	if _, err := os.Stat(nsFile); os.IsNotExist(err) {
		c.Skip("Namespace file is not presented. Skipping")
	}
	nsFromFile, err := ioutil.ReadFile(nsFile)
	c.Assert(err, IsNil)
	c.Assert(nsFromFile, NotNil)
	ns, err := GetControllerNamespace()
	c.Assert(err, IsNil)
	c.Assert(ns, Equals, nsFromFile)
}

func (s *PodInfoSuite) TestGetControllerPodNameFromEnv(c *C) {
	os.Setenv(podNameEnvVar, testPodName)
	podName, err := GetControllerPodName()
	c.Assert(err, IsNil)
	c.Assert(podName, Equals, testPodName)
	os.Unsetenv(podNameEnvVar)
}

func (s *PodInfoSuite) TestGetControllerPodNameFromSystem(c *C) {
	hostname, err := os.Hostname()
	c.Assert(err, IsNil)
	podName, err := GetControllerPodName()
	c.Assert(err, IsNil)
	c.Assert(podName, Equals, hostname)
}

func (s *PodInfoSuite) TestGetControllerServiceAccountFromEnv(c *C) {
	os.Setenv(podSAEnvVar, testPodSA)
	saName, err := GetControllerServiceAccount(fake.NewSimpleClientset())
	c.Assert(err, IsNil)
	c.Assert(saName, Equals, testPodSA)
	os.Unsetenv(testPodSA)
}
