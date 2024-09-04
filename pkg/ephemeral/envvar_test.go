// Copyright 2024 The Kanister Authors.
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

package ephemeral_test

import (
	"os"

	. "gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/kanisterio/kanister/pkg/ephemeral"
	"github.com/kanisterio/kanister/pkg/kube"
)

type EnvVarSuite struct{}

var _ = Suite(&EnvVarSuite{})

func (s *EnvVarSuite) TestOSEnvVarKubePodOptions(c *C) {
	expected := []corev1.EnvVar{
		{
			Name:  "KANISTER_REGISTERED_OS_ENVVAR",
			Value: "1",
		},
	}

	// OS environment variable not set
	var registeredAppliers ephemeral.ApplierList[kube.PodOptions]
	set := ephemeral.OSEnvVar(expected[0].Name)
	registeredAppliers.Register(set.PodOptions)

	var options kube.PodOptions
	registeredAppliers.Apply(&options)
	c.Assert(options.EnvironmentVariables, DeepEquals, []corev1.EnvVar(nil))

	// OS environment variable set
	err := os.Setenv(expected[0].Name, expected[0].Value)
	c.Assert(err, IsNil)
	defer func() {
		err := os.Unsetenv(expected[0].Name)
		c.Assert(err, IsNil)
	}()
	registeredAppliers.Apply(&options)
	c.Assert(options.EnvironmentVariables, DeepEquals, expected)
}

func (s *EnvVarSuite) TestOSEnvVarCoreV1Container(c *C) {
	expected := []corev1.EnvVar{
		{
			Name:  "KANISTER_REGISTERED_OS_ENVVAR",
			Value: "1",
		},
	}

	// OS environment variable not set
	var registeredAppliers ephemeral.ApplierList[corev1.Container]
	set := ephemeral.OSEnvVar(expected[0].Name)
	registeredAppliers.Register(set.Container)

	var options corev1.Container
	registeredAppliers.Apply(&options)
	c.Assert(options.Env, DeepEquals, []corev1.EnvVar(nil))

	// OS environment variable set
	err := os.Setenv(expected[0].Name, expected[0].Value)
	c.Assert(err, IsNil)

	defer func() {
		err := os.Unsetenv(expected[0].Name)
		c.Assert(err, IsNil)
	}()
	registeredAppliers.Apply(&options)
	c.Assert(options.Env, DeepEquals, expected)
}

func (s *EnvVarSuite) TestStaticEnvVarKubePodOptions(c *C) {
	expected := []corev1.EnvVar{
		{
			Name:  "KANISTER_REGISTERED_STATIC_ENVVAR",
			Value: "1",
		},
	}

	var registeredAppliers ephemeral.ApplierList[kube.PodOptions]
	set := ephemeral.StaticEnvVar(expected[0].Name, expected[0].Value)
	registeredAppliers.Register(set.PodOptions)

	var options kube.PodOptions
	registeredAppliers.Apply(&options)
	c.Assert(options.EnvironmentVariables, DeepEquals, expected)
}

func (s *EnvVarSuite) TestRegisteringStaticEnvVarCoreV1Container(c *C) {
	expected := []corev1.EnvVar{
		{
			Name:  "KANISTER_REGISTERED_STATIC_ENVVAR",
			Value: "1",
		},
	}

	var registeredAppliers ephemeral.ApplierList[corev1.Container]
	set := ephemeral.StaticEnvVar(expected[0].Name, expected[0].Value)
	registeredAppliers.Register(set.Container)

	var options corev1.Container
	registeredAppliers.Apply(&options)
	c.Assert(options.Env, DeepEquals, expected)
}
