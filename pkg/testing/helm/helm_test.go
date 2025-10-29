// Copyright 2022 The Kanister Authors.
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

package helm

import (
	"context"
	"fmt"
	"testing"

	"gopkg.in/check.v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/helm"
	"github.com/kanisterio/kanister/pkg/kube"
)

var _ = check.Suite(&HelmTestSuite{})

func Test(t *testing.T) {
	check.TestingT(t)
}

const (
	kanisterName     = "kanister"
	kanisterChartURL = "http://charts.kanister.io"
)

type HelmTestSuite struct {
	deploymentName string
	kubeClient     kubernetes.Interface
	helmApp        HelmApp
}

func (h *HelmTestSuite) SetUpSuite(c *check.C) {
	cfg, err := kube.LoadConfig()
	c.Assert(err, check.IsNil)

	cli, err := kubernetes.NewForConfig(cfg)
	c.Assert(err, check.IsNil)
	h.kubeClient = cli

	h.deploymentName = fmt.Sprintf("%s-%s", kanisterName, "kanister-operator")

	helmValues := map[string]string{
		"bpValidatingWebhook.enabled": "false",
	}

	kanisterApp, err := NewHelmApp(helmValues, kanisterName, "kanister/kanister-operator", kanisterName, "", false)
	c.Assert(err, check.IsNil)

	// add kanister repo
	err = kanisterApp.AddRepo(kanisterName, kanisterChartURL)
	c.Assert(err, check.IsNil)

	h.helmApp = *kanisterApp
}

func (h *HelmTestSuite) TestUpgrade(c *check.C) {
	ctx := context.Background()

	// install released version of kanister
	c.Log("Installing kanister release")
	// TODO: Use manifests to test the helm charts
	_, err := h.helmApp.Install()
	c.Assert(err, check.IsNil)
	// wait for kanister deployment to be ready
	err = kube.WaitOnDeploymentReady(ctx, h.kubeClient, h.helmApp.namespace, h.deploymentName)
	c.Assert(err, check.IsNil)

	c.Log("Upgrading the kanister release with local chart and updated image")
	// upgrade the installed application
	updatedValues := map[string]string{
		"image.tag":                   "v9.99.9-dev",
		"bpValidatingWebhook.enabled": "false",
		"livenessProbe.enabled":       "true",
		"readinessProbe.enabled":      "true",
		"secureDefaultsForJobPods":    "true",
	}

	c.Assert(h.helmApp.Upgrade("../../../helm/kanister-operator", updatedValues), check.IsNil)

	// wait for kanister deployment to be ready
	err = kube.WaitOnDeploymentReady(ctx, h.kubeClient, h.helmApp.namespace, h.deploymentName)
	c.Assert(err, check.IsNil)
}

func (h *HelmTestSuite) TestResourcesFromManifestAfterDryRunHelmInstall(c *check.C) {
	defer func() {
		h.helmApp.dryRun = false
	}()
	c.Log("Installing kanister release - Dry run")
	h.helmApp.dryRun = true
	out, err := h.helmApp.Install()
	c.Assert(err, check.IsNil)
	// Fetch all resources
	resources := helm.ResourcesFromRenderedManifest(out, nil)
	c.Assert(len(resources) > 0, check.Equals, true)
}

// TestSelectedDeploymentAttrFromKanisterHelmDryRunInstall test case does a dry run install of the `kanister` helm chart and validates
// use cases for `nodeSelector` and `toleration` attributes in the helmValues.yaml. This function is specific to `deployment` resource.
func (h *HelmTestSuite) TestSelectedDeploymentAttrFromKanisterHelmDryRunInstall(c *check.C) {
	nodeSelector := map[string]string{
		"selector-key": "selector-value",
	}
	toleration := []corev1.Toleration{
		{
			Key:      "taint-key",
			Operator: corev1.TolerationOpEqual,
			Value:    "taint-value",
			Effect:   corev1.TaintEffectNoSchedule,
		},
	}

	var testCases = []struct {
		testName             string
		helmValues           map[string]string
		expectedNodeSelector map[string]string
		expectedTolerations  []corev1.Toleration
	}{
		{
			testName: "Both nodeSelector and tolerations are present",
			helmValues: map[string]string{
				"bpValidatingWebhook.enabled": "false",
				"nodeSelector.selector-key":   "selector-value",
				"tolerations[0].key":          "taint-key",
				"tolerations[0].operator":     "Equal",
				"tolerations[0].value":        "taint-value",
				"tolerations[0].effect":       "NoSchedule",
			},
			expectedNodeSelector: nodeSelector,
			expectedTolerations:  toleration,
		},
		{
			testName: "Only nodeSelector is present",
			helmValues: map[string]string{
				"bpValidatingWebhook.enabled": "false",
				"nodeSelector.selector-key":   "selector-value",
			},
			expectedNodeSelector: nodeSelector,
			expectedTolerations:  nil,
		},
		{
			testName: "Only tolerations is present",
			helmValues: map[string]string{
				"bpValidatingWebhook.enabled": "false",
				"tolerations[0].key":          "taint-key",
				"tolerations[0].operator":     "Equal",
				"tolerations[0].value":        "taint-value",
				"tolerations[0].effect":       "NoSchedule",
			},
			expectedNodeSelector: nil,
			expectedTolerations:  toleration,
		},
		{
			testName: "Both nodeSelector and tolerations are not present",
			helmValues: map[string]string{
				"bpValidatingWebhook.enabled": "false",
			},
			expectedNodeSelector: nil,
			expectedTolerations:  nil,
		},
	}
	for _, tc := range testCases {
		c.Logf("Test name:%s ", tc.testName)
		defer func() {
			h.helmApp.dryRun = false
		}()
		// Installing kanister release from local kanister-operator - Dry run"
		testApp, err := NewHelmApp(tc.helmValues, kanisterName, "../../../helm/kanister-operator", kanisterName, "", true)
		c.Assert(err, check.IsNil)

		out, err := testApp.Install()
		c.Assert(err, check.IsNil)
		resources := helm.ResourcesFromRenderedManifest(out, func(kind helm.K8sObjectType) bool {
			return kind == helm.K8sObjectTypeDeployment
		})
		c.Assert(len(resources) > 0, check.Equals, true)
		// Take the deployment resources
		deployments, err := helm.K8sObjectsFromRenderedResources[*appsv1.Deployment](resources)
		c.Assert(err, check.IsNil)
		// Use only the required deployment
		var obj = deployments[h.deploymentName]
		c.Assert(obj, check.NotNil)

		c.Assert(obj.Spec.Template.Spec.NodeSelector, check.DeepEquals, tc.expectedNodeSelector)
		c.Assert(obj.Spec.Template.Spec.Tolerations, check.DeepEquals, tc.expectedTolerations)
	}
}

// TestPodRenderingnHelmChart test case does a dry run install of the `kanister` helm chart and validates.
func (h *HelmTestSuite) TestPodRenderingFromHelmChart(c *check.C) {
	podSecurity := corev1.PodSecurityContext{
		RunAsUser:    intPtr(9000),
		FSGroup:      intPtr(9000),
		RunAsNonRoot: boolPtr(true),
		RunAsGroup:   intPtr(9000),
	}

	containerSecurity := corev1.SecurityContext{
		ReadOnlyRootFilesystem:   boolPtr(false),
		AllowPrivilegeEscalation: boolPtr(true),
		Privileged:               boolPtr(true),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
	}

	livenessProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/random_liveness",
				Port: intstr.FromInt(9091),
			},
		},
		InitialDelaySeconds: 5,
		PeriodSeconds:       5,
		FailureThreshold:    5,
		SuccessThreshold:    5,
		TimeoutSeconds:      5,
	}

	readinessProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/random_readiness",
				Port: intstr.FromInt(9091),
			},
		},
		InitialDelaySeconds: 5,
		PeriodSeconds:       5,
		FailureThreshold:    5,
		SuccessThreshold:    5,
		TimeoutSeconds:      5,
	}

	var testCases = []struct {
		testName                  string
		helmValues                map[string]string
		expectedPodSecurity       *corev1.PodSecurityContext
		expectedContainerSecurity *corev1.SecurityContext
		expectedLivenessProbe     *corev1.Probe
		expectedReadinessProbe    *corev1.Probe
	}{
		{
			testName: "Pod and Container security context are set",
			helmValues: map[string]string{
				"containerSecurityContext.capabilities.drop[0]":     "ALL",
				"containerSecurityContext.privileged":               "true",
				"containerSecurityContext.allowPrivilegeEscalation": "true",
				"containerSecurityContext.readOnlyRootFilesystem":   "false",
				"podSecurityContext.runAsUser":                      "9000",
				"podSecurityContext.fsGroup":                        "9000",
				"podSecurityContext.runAsNonRoot":                   "true",
				"podSecurityContext.runAsGroup":                     "9000",
			},
			expectedPodSecurity:       &podSecurity,
			expectedContainerSecurity: &containerSecurity,
		},
		{
			testName: "Only Container security context is getting overwritten",
			helmValues: map[string]string{
				"containerSecurityContext.capabilities.drop[0]":     "ALL",
				"containerSecurityContext.privileged":               "true",
				"containerSecurityContext.allowPrivilegeEscalation": "true",
				"containerSecurityContext.readOnlyRootFilesystem":   "false",
			},
			expectedContainerSecurity: &containerSecurity,
		},
		{
			testName: "Only Pod security context is getting overwritten",
			helmValues: map[string]string{
				"podSecurityContext.runAsUser":    "9000",
				"podSecurityContext.fsGroup":      "9000",
				"podSecurityContext.runAsNonRoot": "true",
				"podSecurityContext.runAsGroup":   "9000",
			},
			expectedPodSecurity: &podSecurity,
		},
		{
			testName: "Disable liveness and readiness probes",
			helmValues: map[string]string{
				"livenessProbe.enabled":  "false",
				"readinessProbe.enabled": "false",
			},
		},
		{
			testName: "Liveness and readiness probes is getting overwritten",
			helmValues: map[string]string{
				"healthCheckPort":                    "9091",
				"livenessProbe.enabled":              "true",
				"livenessProbe.httpGet.path":         "/random_liveness",
				"livenessProbe.initialDelaySeconds":  "5",
				"livenessProbe.periodSeconds":        "5",
				"livenessProbe.failureThreshold":     "5",
				"livenessProbe.successThreshold":     "5",
				"livenessProbe.timeoutSeconds":       "5",
				"readinessProbe.enabled":             "true",
				"readinessProbe.httpGet.path":        "/random_readiness",
				"readinessProbe.initialDelaySeconds": "5",
				"readinessProbe.periodSeconds":       "5",
				"readinessProbe.failureThreshold":    "5",
				"readinessProbe.successThreshold":    "5",
				"readinessProbe.timeoutSeconds":      "5",
			},
			expectedLivenessProbe:  livenessProbe,
			expectedReadinessProbe: readinessProbe,
		},
	}

	for _, tc := range testCases {
		c.Logf("Test name: %s", tc.testName)
		defer func() {
			h.helmApp.dryRun = false
		}()

		testApp, err := NewHelmApp(tc.helmValues, kanisterName, "../../../helm/kanister-operator", kanisterName, "", true)
		c.Assert(err, check.IsNil)

		out, err := testApp.Install()
		c.Assert(err, check.IsNil)

		resources := helm.ResourcesFromRenderedManifest(out, func(kind helm.K8sObjectType) bool {
			return kind == helm.K8sObjectTypeDeployment
		})
		c.Assert(len(resources), check.Equals, 1)

		deployments, err := helm.K8sObjectsFromRenderedResources[*appsv1.Deployment](resources)
		c.Assert(err, check.IsNil)

		var obj = deployments[h.deploymentName]
		c.Assert(obj, check.NotNil)

		if tc.expectedPodSecurity != nil {
			c.Assert(obj.Spec.Template.Spec.SecurityContext, check.DeepEquals, tc.expectedPodSecurity)
		}

		if tc.expectedContainerSecurity != nil {
			c.Assert(obj.Spec.Template.Spec.Containers[0].SecurityContext, check.DeepEquals, tc.expectedContainerSecurity)
		}

		if enabled, ok := tc.helmValues["livenessProbe.enabled"]; ok {
			if enabled == "true" {
				c.Assert(obj.Spec.Template.Spec.Containers[0].LivenessProbe, check.NotNil)
				c.Assert(obj.Spec.Template.Spec.Containers[0].LivenessProbe, check.DeepEquals, tc.expectedLivenessProbe)
			} else {
				c.Assert(obj.Spec.Template.Spec.Containers[0].LivenessProbe, check.IsNil)
			}
		}

		if enabled, ok := tc.helmValues["readinessProbe.enabled"]; ok {
			if enabled == "true" {
				c.Assert(obj.Spec.Template.Spec.Containers[0].ReadinessProbe, check.NotNil)
				c.Assert(obj.Spec.Template.Spec.Containers[0].ReadinessProbe, check.DeepEquals, tc.expectedReadinessProbe)
			} else {
				c.Assert(obj.Spec.Template.Spec.Containers[0].ReadinessProbe, check.IsNil)
			}
		}
	}
}

// TestPodAnnotationsFromKanisterHelmDryRunInstall test case does a dry run install of the `kanister` helm chart and validates
// use cases for `podAnnotations` attributes in the helmValues.yaml. This function is specific to `deployment` resource.
func (h *HelmTestSuite) TestPodAnnotationsFromKanisterHelmDryRunInstall(c *check.C) {
	expectedAnnotations := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	var testCases = []struct {
		testName            string
		helmValues          map[string]string
		expectedAnnotations map[string]string
	}{
		{
			testName: "Pod annotations are present",
			helmValues: map[string]string{
				"bpValidatingWebhook.enabled": "false",
				"podAnnotations.key1":         "value1",
				"podAnnotations.key2":         "value2",
			},
			expectedAnnotations: expectedAnnotations,
		},
		{
			testName: "No pod annotations",
			helmValues: map[string]string{
				"bpValidatingWebhook.enabled": "false",
			},
			expectedAnnotations: nil,
		},
	}

	for _, tc := range testCases {
		c.Logf("Test name:%s ", tc.testName)
		defer func() {
			h.helmApp.dryRun = false
		}()
		// Installing kanister release from local kanister-operator - Dry run"
		testApp, err := NewHelmApp(tc.helmValues, kanisterName, "../../../helm/kanister-operator", kanisterName, "", true)
		c.Assert(err, check.IsNil)

		out, err := testApp.Install()
		c.Assert(err, check.IsNil)
		resources := helm.ResourcesFromRenderedManifest(out, func(kind helm.K8sObjectType) bool {
			return kind == helm.K8sObjectTypeDeployment
		})
		c.Assert(len(resources) > 0, check.Equals, true)
		// Take the deployment resources
		deployments, err := helm.K8sObjectsFromRenderedResources[*appsv1.Deployment](resources)
		c.Assert(err, check.IsNil)
		// Use only the required deployment
		var obj = deployments[h.deploymentName]
		c.Assert(obj, check.NotNil)

		c.Assert(obj.Spec.Template.ObjectMeta.Annotations, check.DeepEquals, tc.expectedAnnotations)
	}
}

func (h *HelmTestSuite) TestSecureDefaultsEnvVariable(c *check.C) {
	var testCases = []struct {
		testName               string
		helmValues             map[string]string
		expectedSecureDefaults bool
		expectedEnvVar         corev1.EnvVar
	}{
		{
			testName: "SecureDefaultsForJobPods enabled and environment variable set",
			helmValues: map[string]string{
				"secureDefaultsForJobPods": "true",
			},
			expectedEnvVar: corev1.EnvVar{
				Name:  "SECURE_DEFAULTS_FOR_JOB_PODS",
				Value: "true",
			},
		},
	}

	for _, tc := range testCases {
		c.Logf("Test name: %s", tc.testName)
		defer func() {
			h.helmApp.dryRun = false
		}()

		testApp, err := NewHelmApp(tc.helmValues, kanisterName, "../../../helm/kanister-operator", kanisterName, "", true)
		c.Assert(err, check.IsNil)

		out, err := testApp.Install()
		c.Assert(err, check.IsNil)

		resources := helm.ResourcesFromRenderedManifest(out, func(kind helm.K8sObjectType) bool {
			return kind == helm.K8sObjectTypeDeployment
		})
		c.Assert(len(resources), check.Equals, 1)

		deployments, err := helm.K8sObjectsFromRenderedResources[*appsv1.Deployment](resources)
		c.Assert(err, check.IsNil)

		var obj = deployments[h.deploymentName]
		c.Assert(obj, check.NotNil)

		// Verify secureDefaultsForJobPods flag when enable adds environment variable.
		if enabled, ok := tc.helmValues["secureDefaultsForJobPods"]; ok {
			if enabled == "true" {
				listEnvs := obj.Spec.Template.Spec.Containers[0].Env
				found := false
				for _, env := range listEnvs {
					if env.Name == tc.expectedEnvVar.Name && env.Value == tc.expectedEnvVar.Value {
						found = true
						break
					}
				}
				c.Assert(found, check.Equals, true, check.Commentf("Environment variable %s with value %s not found", tc.expectedEnvVar.Name, tc.expectedEnvVar.Value))
			}
		}
	}
}

func boolPtr(b bool) *bool {
	return &b
}

func intPtr(i int64) *int64 {
	return &i
}

func (h *HelmTestSuite) TearDownSuite(c *check.C) {
	c.Log("Uninstalling chart")
	err := h.helmApp.Uninstall()
	c.Assert(err, check.IsNil)
	// Uninstall doesn't delete namespace, delete namespace separately
	c.Assert(h.kubeClient.CoreV1().Namespaces().Delete(context.Background(), h.helmApp.namespace, metav1.DeleteOptions{}), check.IsNil)
}
