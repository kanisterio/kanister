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

	. "gopkg.in/check.v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/helm"
	"github.com/kanisterio/kanister/pkg/kube"
)

var _ = Suite(&HelmTestSuite{})

func Test(t *testing.T) {
	TestingT(t)
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

func (h *HelmTestSuite) SetUpSuite(c *C) {
	cfg, err := kube.LoadConfig()
	c.Assert(err, IsNil)

	cli, err := kubernetes.NewForConfig(cfg)
	c.Assert(err, IsNil)
	h.kubeClient = cli

	h.deploymentName = fmt.Sprintf("%s-%s", kanisterName, "kanister-operator")

	helmValues := map[string]string{
		"bpValidatingWebhook.enabled": "false",
	}

	kanisterApp, err := NewHelmApp(helmValues, kanisterName, "kanister/kanister-operator", kanisterName, "", false)
	c.Assert(err, IsNil)

	// add kanister repo
	err = kanisterApp.AddRepo(kanisterName, kanisterChartURL)
	c.Assert(err, IsNil)

	h.helmApp = *kanisterApp
}

func (h *HelmTestSuite) TestUpgrade(c *C) {
	ctx := context.Background()

	// install released version of kanister
	c.Log("Installing kanister release")
	// TODO: Use manifests to test the helm charts
	_, err := h.helmApp.Install()
	c.Assert(err, IsNil)
	// wait for kanister deployment to be ready
	err = kube.WaitOnDeploymentReady(ctx, h.kubeClient, h.helmApp.namespace, h.deploymentName)
	c.Assert(err, IsNil)

	c.Log("Upgrading the kanister release with local chart and updated image")
	// upgrade the installed application
	updatedValues := map[string]string{
		"image.tag":                   "v9.99.9-dev",
		"bpValidatingWebhook.enabled": "false",
	}
	c.Assert(h.helmApp.Upgrade("../../../helm/kanister-operator", updatedValues), IsNil)

	// wait for kanister deployment to be ready
	err = kube.WaitOnDeploymentReady(ctx, h.kubeClient, h.helmApp.namespace, h.deploymentName)
	c.Assert(err, IsNil)
}

func (h *HelmTestSuite) TestDryRunInstall(c *C) {
	defer func() {
		h.helmApp.dryRun = false
	}()
	c.Log("Installing kanister release - Dry run")
	h.helmApp.dryRun = true
	out, err := h.helmApp.Install()
	c.Assert(err, IsNil)
	releaseName := helm.ParseReleaseNameFromHelmStatus(out)
	c.Assert(releaseName, Equals, kanisterName)
}

func (h *HelmTestSuite) TestComponentsFromManifestAfterDryRunHelmInstall(c *C) {
	defer func() {
		h.helmApp.dryRun = false
	}()
	c.Log("Installing kanister release - Dry run")
	h.helmApp.dryRun = true
	out, err := h.helmApp.Install()
	c.Assert(err, IsNil)
	components := helm.ComponentsFromManifest(out)
	/*
		Following are components from kanister include :
		1. kanister-kanister-operator (serviceaccount)
		2. kanister-kanister-operator-cluster-role (clusterrole)
		3. kanister-kanister-operator-edit-role (clusterrolebinding)
		4. kanister-kanister-operator-cr-role (clusterrolebinding)
		5. kanister-kanister-operator-pv-provisioner (clusterrolebinding)
		6. kanister-kanister-operator (service)
		7. kanister-kanister-operator (deployment)
	*/
	c.Assert(len(components), Equals, 7)
}

func (h *HelmTestSuite) TearDownSuite(c *C) {
	c.Log("Uninstalling chart")
	err := h.helmApp.Uninstall()
	c.Assert(err, IsNil)
	// Uninstall doesn't delete namespace, delete namespace separately
	c.Assert(h.kubeClient.CoreV1().Namespaces().Delete(context.Background(), h.helmApp.namespace, metav1.DeleteOptions{}), IsNil)
}
