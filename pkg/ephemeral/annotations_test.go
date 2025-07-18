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

	"gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/kanisterio/kanister/pkg/ephemeral"
	"github.com/kanisterio/kanister/pkg/kube"
)

type AnnotationsSuite struct{}

var _ = check.Suite(&AnnotationsSuite{})

const annotationsEnvVar = "LABEL_ENV_VAR"
const annotationsEnvVarValues = `{"acs.amadeus.com/securityZone":"app","app.kubernetes.io/component":"kasten"}`

func (s *AnnotationsSuite) TestLabelsFromEnvVarKubePodOptions(c *check.C) {
	expected := map[string]string{
		"acs.amadeus.com/securityZone": "app",
		"app.kubernetes.io/component":  "kasten",
	}

	// OS environment variable not set
	var registeredAppliers ephemeral.ApplierList[kube.PodOptions]
	set := ephemeral.AnnotationsFromEnvVar(annotationsEnvVar)
	registeredAppliers.Register(set.PodOptions)

	var options kube.PodOptions
	err := registeredAppliers.Apply(&options)
	c.Assert(err, check.IsNil)
	c.Assert(options.Annotations, check.DeepEquals, map[string]string(nil))

	// OS environment variable set
	c.Setenv(annotationsEnvVar, annotationsEnvVarValues)
	defer func() {
		err := os.Unsetenv(annotationsEnvVar)
		c.Assert(err, check.IsNil)
	}()
	err = registeredAppliers.Apply(&options)
	c.Assert(err, check.IsNil)
	c.Assert(options.Annotations, check.DeepEquals, expected)
}

func (s *AnnotationsSuite) TestLabelsFromEnvVarCoreV1Pod(c *check.C) {
	expected := map[string]string{
		"acs.amadeus.com/securityZone": "app",
		"app.kubernetes.io/component":  "kasten",
	}

	// OS environment variable not set
	var registeredAppliers ephemeral.ApplierList[corev1.Pod]
	set := ephemeral.AnnotationsFromEnvVar(annotationsEnvVar)
	registeredAppliers.Register(set.Pod)

	var options corev1.Pod
	registeredAppliers.Apply(&options)
	c.Assert(options.Annotations, check.DeepEquals, map[string]string(nil))

	// OS environment variable set
	err := os.Setenv(annotationsEnvVar, annotationsEnvVarValues)
	c.Assert(err, check.IsNil)
	defer func() {
		err := os.Unsetenv(annotationsEnvVar)
		c.Assert(err, check.IsNil)
	}()
	registeredAppliers.Apply(&options)
	c.Assert(options.Annotations, check.DeepEquals, expected)
}
