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

package kube

import (
	"os"

	. "gopkg.in/check.v1"

	"k8s.io/client-go/kubernetes/fake"
)

type PodInfoSuite struct{}

var _ = Suite(&PodInfoSuite{})

const testPodNamespace = "test-pod-namespace"
const testPodName = "test-pod-name"
const testPodSA = "test-pod-sa"

func (s *PodInfoSuite) TestGetControllerNamespaceFromEnv(c *C) {
	os.Setenv(PodNSEnvVar, testPodNamespace)
	ns, err := GetControllerNamespace()
	c.Assert(err, IsNil)
	c.Assert(ns, Equals, testPodNamespace)
	os.Unsetenv(PodNSEnvVar)
}

func (s *PodInfoSuite) TestGetControllerNamespaceFromFile(c *C) {
	if _, err := os.Stat(nsFile); os.IsNotExist(err) {
		c.Skip("Namespace file is not presented. Skipping")
	}
	nsFromFile, err := os.ReadFile(nsFile)
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
	os.Setenv(PodSAEnvVar, testPodSA)
	saName, err := GetControllerServiceAccount(fake.NewSimpleClientset())
	c.Assert(err, IsNil)
	c.Assert(saName, Equals, testPodSA)
	os.Unsetenv(testPodSA)
}
