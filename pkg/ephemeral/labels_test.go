/*
Copyright 2025 by contributors to the Kanister project.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ephemeral_test

import (
	"os"

	"github.com/kanisterio/kanister/pkg/ephemeral"
	"github.com/kanisterio/kanister/pkg/kube"
	"gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
)

type LabelsSuite struct{}

var _ = check.Suite(&LabelsSuite{})

const labelsEnvVar = "LABEL_ENV_VAR"
const invalidLabelsEnvVar = "INVALID_LABELS_ENV_VAR"
const labelsEnvVarValues = `{"acs.amadeus.com/securityZone":"app","app.kubernetes.io/component":"kasten"}`

func (s *LabelsSuite) TestLabelsFromEnvVarKubePodOptionsEnvVarUnset(c *check.C) {
	// OS environment variable not set
	var registeredAppliers ephemeral.ApplierList[kube.PodOptions]
	set := ephemeral.LabelsFromEnvVar(labelsEnvVar)
	registeredAppliers.Register(set.PodOptions)

	var options kube.PodOptions
	err := registeredAppliers.Apply(&options)
	c.Assert(err, check.IsNil)
	c.Assert(options.Labels, check.DeepEquals, map[string]string(nil))
}

func (s *LabelsSuite) TestLabelsFromEnvVarKubePodOptionsEnvVarSet(c *check.C) {
	expected := map[string]string{
		"acs.amadeus.com/securityZone": "app",
		"app.kubernetes.io/component":  "kasten",
	}

	// OS environment variable not set
	var registeredAppliers ephemeral.ApplierList[kube.PodOptions]
	set := ephemeral.LabelsFromEnvVar(labelsEnvVar)
	registeredAppliers.Register(set.PodOptions)

	// OS environment variable set
	err := os.Setenv(labelsEnvVar, labelsEnvVarValues)
	c.Assert(err, check.IsNil)
	defer func() {
		err := os.Unsetenv(labelsEnvVar)
		c.Assert(err, check.IsNil)
	}()
	var options kube.PodOptions
	err = registeredAppliers.Apply(&options)
	c.Assert(err, check.IsNil)
	c.Assert(options.Labels, check.DeepEquals, expected)
}

func (s *LabelsSuite) TestLabelsFromEnvVarCoreV1PodEnvVarUnset(c *check.C) {
	// OS environment variable not set
	var registeredAppliers ephemeral.ApplierList[corev1.Pod]
	set := ephemeral.LabelsFromEnvVar(labelsEnvVar)
	registeredAppliers.Register(set.Pod)

	var options corev1.Pod
	registeredAppliers.Apply(&options)
	c.Assert(options.Labels, check.DeepEquals, map[string]string(nil))
}

func (s *LabelsSuite) TestLabelsFromEnvVarCoreV1PodEnvVarSet(c *check.C) {
	expected := map[string]string{
		"acs.amadeus.com/securityZone": "app",
		"app.kubernetes.io/component":  "kasten",
	}

	// OS environment variable not set
	var registeredAppliers ephemeral.ApplierList[corev1.Pod]
	set := ephemeral.LabelsFromEnvVar(labelsEnvVar)
	registeredAppliers.Register(set.Pod)

	// OS environment variable set
	err := os.Setenv(labelsEnvVar, labelsEnvVarValues)
	c.Assert(err, check.IsNil)
	defer func() {
		err := os.Unsetenv(labelsEnvVar)
		c.Assert(err, check.IsNil)
	}()
	var options corev1.Pod
	registeredAppliers.Apply(&options)
	c.Assert(options.Labels, check.DeepEquals, expected)
}

func (s *LabelsSuite) TestLabelsFromEnvVarPodOptionsInvalidJSON(c *check.C) {
	err := os.Setenv(invalidJSONEnvVar, "{invalid-json}")
	c.Assert(err, check.IsNil)
	defer func() {
		err := os.Unsetenv(invalidJSONEnvVar)
		c.Assert(err, check.IsNil)
	}()

	var registeredAppliers ephemeral.ApplierList[kube.PodOptions]
	set := ephemeral.LabelsFromEnvVar(invalidJSONEnvVar)
	registeredAppliers.Register(set.PodOptions)

	var options kube.PodOptions
	err = registeredAppliers.Apply(&options)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Matches, ".*invalid character.*")
}

func (s *LabelsSuite) TestLabelsFromEnvVarPodOptionsInvalidLabels(c *check.C) {
	invalidLabels := `{"/badkey": "value"}`
	err := os.Setenv(invalidLabelsEnvVar, invalidLabels)
	c.Assert(err, check.IsNil)
	defer func() {
		err := os.Unsetenv(invalidLabelsEnvVar)
		c.Assert(err, check.IsNil)
	}()

	var registeredAppliers ephemeral.ApplierList[kube.PodOptions]
	set := ephemeral.LabelsFromEnvVar(invalidLabelsEnvVar)
	registeredAppliers.Register(set.PodOptions)

	var options kube.PodOptions
	err = registeredAppliers.Apply(&options)
	c.Assert(err, check.NotNil)
}

func (s *LabelsSuite) TestLabelsFromEnvVarCoreV1PodInvalidJSON(c *check.C) {
	err := os.Setenv(invalidJSONEnvVar, "{invalid-json}")
	c.Assert(err, check.IsNil)
	defer func() {
		err := os.Unsetenv(invalidJSONEnvVar)
		c.Assert(err, check.IsNil)
	}()

	var registeredAppliers ephemeral.ApplierList[corev1.Pod]
	set := ephemeral.LabelsFromEnvVar(invalidJSONEnvVar)
	registeredAppliers.Register(set.Pod)

	var options corev1.Pod
	err = registeredAppliers.Apply(&options)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Matches, ".*invalid character.*")
}

func (s *LabelsSuite) TestLabelsFromEnvVarCoreV1PodInvalidLabels(c *check.C) {
	invalidLabels := `{"/badkey": "value"}`
	err := os.Setenv(invalidLabelsEnvVar, invalidLabels)
	c.Assert(err, check.IsNil)
	defer func() {
		err := os.Unsetenv(invalidLabelsEnvVar)
		c.Assert(err, check.IsNil)
	}()

	var registeredAppliers ephemeral.ApplierList[corev1.Pod]
	set := ephemeral.LabelsFromEnvVar(invalidLabelsEnvVar)
	registeredAppliers.Register(set.Pod)

	var options corev1.Pod
	err = registeredAppliers.Apply(&options)
	c.Assert(err, check.NotNil)
}

func (s *LabelsSuite) TestStaticLabelsKubePodOptions(c *check.C) {
	expected := map[string]string{
		"acs.amadeus.com/securityZone": "app",
		"app.kubernetes.io/component":  "kasten",
	}

	var registeredAppliers ephemeral.ApplierList[kube.PodOptions]
	set := ephemeral.StaticLabels(expected)
	registeredAppliers.Register(set.PodOptions)

	var options kube.PodOptions
	err := registeredAppliers.Apply(&options)
	c.Assert(err, check.IsNil)
	c.Assert(options.Labels, check.DeepEquals, expected)
}

func (s *LabelsSuite) TestStaticLabelsCoreV1Pod(c *check.C) {
	expected := map[string]string{
		"acs.amadeus.com/securityZone": "app",
		"app.kubernetes.io/component":  "kasten",
	}

	var registeredAppliers ephemeral.ApplierList[corev1.Pod]
	set := ephemeral.StaticLabels(expected)
	registeredAppliers.Register(set.Pod)

	var options corev1.Pod
	err := registeredAppliers.Apply(&options)
	c.Assert(err, check.IsNil)
	c.Assert(options.Labels, check.DeepEquals, expected)
}

func (s *LabelsSuite) TestStaticLabelsOSEnvVarKubePodOptionsEnvVarUnset(c *check.C) {
	labels := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	envVars := []string{"ENV_VAR_1", "ENV_VAR_2"}

	var registeredAppliers ephemeral.ApplierList[kube.PodOptions]
	set := ephemeral.StaticLabelsOSEnvVar(labels, envVars...)
	registeredAppliers.Register(set.PodOptions)

	var options kube.PodOptions
	err := registeredAppliers.Apply(&options)
	c.Assert(err, check.IsNil)
	c.Assert(options.Labels, check.DeepEquals, map[string]string(nil))
}

func (s *LabelsSuite) TestStaticLabelsOSEnvVarKubePodOptionsEnvVarSet(c *check.C) {
	expected := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	envVars := []string{"ENV_VAR_1", "ENV_VAR_2"}

	// Set environment variables
	err := os.Setenv("ENV_VAR_1", "value")
	c.Assert(err, check.IsNil)
	err = os.Setenv("ENV_VAR_2", "value")
	c.Assert(err, check.IsNil)
	defer func() {
		err := os.Unsetenv("ENV_VAR_1")
		c.Assert(err, check.IsNil)
		err = os.Unsetenv("ENV_VAR_2")
		c.Assert(err, check.IsNil)
	}()

	var registeredAppliers ephemeral.ApplierList[kube.PodOptions]
	set := ephemeral.StaticLabelsOSEnvVar(expected, envVars...)
	registeredAppliers.Register(set.PodOptions)

	var options kube.PodOptions
	err = registeredAppliers.Apply(&options)
	c.Assert(err, check.IsNil)
	c.Assert(options.Labels, check.DeepEquals, expected)
}

func (s *LabelsSuite) TestStaticLabelsOSEnvVarCoreV1PodEnvVarUnset(c *check.C) {
	labels := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	envVars := []string{"ENV_VAR_1", "ENV_VAR_2"}

	var registeredAppliers ephemeral.ApplierList[corev1.Pod]
	set := ephemeral.StaticLabelsOSEnvVar(labels, envVars...)
	registeredAppliers.Register(set.Pod)

	var options corev1.Pod
	err := registeredAppliers.Apply(&options)
	c.Assert(err, check.IsNil)
	c.Assert(options.Labels, check.DeepEquals, map[string]string(nil))
}

func (s *LabelsSuite) TestStaticLabelsOSEnvVarCoreV1PodEnvVarSet(c *check.C) {
	expected := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	envVars := []string{"ENV_VAR_1", "ENV_VAR_2"}

	// Set environment variables
	err := os.Setenv("ENV_VAR_1", "value")
	c.Assert(err, check.IsNil)
	err = os.Setenv("ENV_VAR_2", "value")
	c.Assert(err, check.IsNil)
	defer func() {
		err := os.Unsetenv("ENV_VAR_1")
		c.Assert(err, check.IsNil)
		err = os.Unsetenv("ENV_VAR_2")
		c.Assert(err, check.IsNil)
	}()

	var registeredAppliers ephemeral.ApplierList[corev1.Pod]
	set := ephemeral.StaticLabelsOSEnvVar(expected, envVars...)
	registeredAppliers.Register(set.Pod)

	var options corev1.Pod
	err = registeredAppliers.Apply(&options)
	c.Assert(err, check.IsNil)
	c.Assert(options.Labels, check.DeepEquals, expected)
}
