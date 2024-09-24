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

func (h *HelmTestSuite) TearDownSuite(c *check.C) {
	c.Log("Uninstalling chart")
	err := h.helmApp.Uninstall()
	c.Assert(err, check.IsNil)
	// Uninstall doesn't delete namespace, delete namespace separately
	c.Assert(h.kubeClient.CoreV1().Namespaces().Delete(context.Background(), h.helmApp.namespace, metav1.DeleteOptions{}), check.IsNil)
}
