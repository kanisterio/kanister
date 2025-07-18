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

	"github.com/kanisterio/kanister/pkg/ephemeral"
	"github.com/kanisterio/kanister/pkg/kube"
	"gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
)

type AnnotationsSuite struct{}

var _ = check.Suite(&AnnotationsSuite{})

const annotationsEnvVar = "ANNOTATIONS_ENV_VAR"
const invalidAnnotationsEnvVar = "INVALID_ANNOTATIONS_ENV_VAR"
const invalidJSONEnvVar = "INVALID_JSON_ENV_VAR"
const annotationsEnvVarValues = `{"acs.amadeus.com/securityZone":"app","app.kubernetes.io/component":"kasten"}`

func (s *AnnotationsSuite) TestAnnotationsFromEnvVarKubePodOptionsEnvVarUnset(c *check.C) {
	// OS environment variable not set
	var registeredAppliers ephemeral.ApplierList[kube.PodOptions]
	set := ephemeral.AnnotationsFromEnvVar(annotationsEnvVar)
	registeredAppliers.Register(set.PodOptions)

	var options kube.PodOptions
	err := registeredAppliers.Apply(&options)
	c.Assert(err, check.IsNil)
	c.Assert(options.Annotations, check.DeepEquals, map[string]string(nil))
}

func (s *AnnotationsSuite) TestAnnotationsFromEnvVarKubePodOptionsEnvVarSet(c *check.C) {
	expected := map[string]string{
		"acs.amadeus.com/securityZone": "app",
		"app.kubernetes.io/component":  "kasten",
	}

	var registeredAppliers ephemeral.ApplierList[kube.PodOptions]
	set := ephemeral.AnnotationsFromEnvVar(annotationsEnvVar)
	registeredAppliers.Register(set.PodOptions)

	// OS environment variable set
	os.Setenv(annotationsEnvVar, annotationsEnvVarValues)
	defer func() {
		err := os.Unsetenv(annotationsEnvVar)
		c.Assert(err, check.IsNil)
	}()
	var options kube.PodOptions
	err := registeredAppliers.Apply(&options)
	c.Assert(err, check.IsNil)
	c.Assert(options.Annotations, check.DeepEquals, expected)
}

func (s *AnnotationsSuite) TestAnnotationsFromEnvVarCoreV1PodEnvVarUnset(c *check.C) {
	// OS environment variable not set
	var registeredAppliers ephemeral.ApplierList[corev1.Pod]
	set := ephemeral.AnnotationsFromEnvVar(annotationsEnvVar)
	registeredAppliers.Register(set.Pod)

	var options corev1.Pod
	registeredAppliers.Apply(&options)
	c.Assert(options.Annotations, check.DeepEquals, map[string]string(nil))
}

func (s *AnnotationsSuite) TestAnnotationsFromEnvVarCoreV1PodEnvVarSet(c *check.C) {
	expected := map[string]string{
		"acs.amadeus.com/securityZone": "app",
		"app.kubernetes.io/component":  "kasten",
	}
	var registeredAppliers ephemeral.ApplierList[corev1.Pod]
	set := ephemeral.AnnotationsFromEnvVar(annotationsEnvVar)
	registeredAppliers.Register(set.Pod)

	err := os.Setenv(annotationsEnvVar, annotationsEnvVarValues)
	c.Assert(err, check.IsNil)
	defer func() {
		err := os.Unsetenv(annotationsEnvVar)
		c.Assert(err, check.IsNil)
	}()
	var options corev1.Pod
	registeredAppliers.Apply(&options)
	c.Assert(options.Annotations, check.DeepEquals, expected)
}

func (s *AnnotationsSuite) TestAnnotationsFromEnvVarPodOptionsInvalidJSON(c *check.C) {
	err := os.Setenv(invalidJSONEnvVar, "{invalid-json}")
	c.Assert(err, check.IsNil)
	defer func() {
		err := os.Unsetenv(invalidJSONEnvVar)
		c.Assert(err, check.IsNil)
	}()

	var registeredAppliers ephemeral.ApplierList[kube.PodOptions]
	set := ephemeral.AnnotationsFromEnvVar(invalidJSONEnvVar)
	registeredAppliers.Register(set.PodOptions)

	var options kube.PodOptions
	err = registeredAppliers.Apply(&options)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Matches, ".*invalid character.*")
}

func (s *AnnotationsSuite) TestAnnotationsFromEnvVarPodOptionsInvalidAnnotations(c *check.C) {
	invalidAnnotations := `{"/key": "value"}`
	err := os.Setenv("INVALID_ANNOTATIONS_ENV_VAR", invalidAnnotations)
	c.Assert(err, check.IsNil)
	defer func() {
		err := os.Unsetenv("INVALID_ANNOTATIONS_ENV_VAR")
		c.Assert(err, check.IsNil)
	}()

	var registeredAppliers ephemeral.ApplierList[kube.PodOptions]
	set := ephemeral.AnnotationsFromEnvVar("INVALID_ANNOTATIONS_ENV_VAR")
	registeredAppliers.Register(set.PodOptions)

	var options kube.PodOptions
	err = registeredAppliers.Apply(&options)
	c.Assert(err, check.NotNil)
}

func (s *AnnotationsSuite) TestAnnotationsFromEnvVarInvalidJSONCoreV1Pod(c *check.C) {
	err := os.Setenv("INVALID_JSON_ENV_VAR", "{invalid-json}")
	c.Assert(err, check.IsNil)
	defer func() {
		err := os.Unsetenv("INVALID_JSON_ENV_VAR")
		c.Assert(err, check.IsNil)
	}()

	var registeredAppliers ephemeral.ApplierList[corev1.Pod]
	set := ephemeral.AnnotationsFromEnvVar("INVALID_JSON_ENV_VAR")
	registeredAppliers.Register(set.Pod)

	var options corev1.Pod
	err = registeredAppliers.Apply(&options)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Matches, ".*invalid character.*")
}

func (s *AnnotationsSuite) TestAnnotationsFromEnvVarInvalidAnnotationsCoreV1Pod(c *check.C) {
	invalidAnnotations := `{"/key": "value"}`
	err := os.Setenv(invalidAnnotationsEnvVar, invalidAnnotations)
	c.Assert(err, check.IsNil)
	defer func() {
		err := os.Unsetenv(invalidAnnotationsEnvVar)
		c.Assert(err, check.IsNil)
	}()

	var registeredAppliers ephemeral.ApplierList[corev1.Pod]
	set := ephemeral.AnnotationsFromEnvVar(invalidAnnotationsEnvVar)
	registeredAppliers.Register(set.Pod)

	var options corev1.Pod
	err = registeredAppliers.Apply(&options)
	c.Assert(err, check.NotNil)
}
