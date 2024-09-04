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
	err := os.Setenv(PodNSEnvVar, testPodNamespace)
	c.Assert(err, IsNil)
	ns, err := GetControllerNamespace()
	c.Assert(err, IsNil)
	c.Assert(ns, Equals, testPodNamespace)
	err = os.Unsetenv(PodNSEnvVar)
	c.Assert(err, IsNil)
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
	err := os.Setenv(podNameEnvVar, testPodName)
	c.Assert(err, IsNil)

	podName, err := GetControllerPodName()
	c.Assert(err, IsNil)
	c.Assert(podName, Equals, testPodName)
	err = os.Unsetenv(podNameEnvVar)
	c.Assert(err, IsNil)
}

func (s *PodInfoSuite) TestGetControllerPodNameFromSystem(c *C) {
	hostname, err := os.Hostname()
	c.Assert(err, IsNil)
	podName, err := GetControllerPodName()
	c.Assert(err, IsNil)
	c.Assert(podName, Equals, hostname)
}

func (s *PodInfoSuite) TestGetControllerServiceAccountFromEnv(c *C) {
	err := os.Setenv(PodSAEnvVar, testPodSA)
	c.Assert(err, IsNil)

	saName, err := GetControllerServiceAccount(fake.NewSimpleClientset())
	c.Assert(err, IsNil)
	c.Assert(saName, Equals, testPodSA)
	err = os.Unsetenv(testPodSA)
	c.Assert(err, IsNil)
}
