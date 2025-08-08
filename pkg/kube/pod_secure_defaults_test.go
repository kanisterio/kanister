// Copyright 2025 The Kanister Authors.
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

//go:build !unit
// +build !unit

package kube

import (
	"context"
	"os"

	"gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/consts"
)

type PodSuiteSecureDefaults struct {
	cli       kubernetes.Interface
	namespace string
}

type testcase struct {
	description string
	podOptions  *PodOptions
}

const (
	dummyServiceAccount  = "dummy-sa"
	secureDefaultsEnvVar = "SECURE_DEFAULTS_FOR_JOB_PODS"
)

var _ = check.Suite(&PodSuiteSecureDefaults{})

func (s *PodSuiteSecureDefaults) SetUpSuite(c *check.C) {
	var err error
	s.cli, err = NewClient()
	c.Assert(err, check.IsNil)
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "podtestsecuredefaults-",
		},
	}
	ns, err = s.cli.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)
	s.namespace = ns.Name

	err = os.Setenv("POD_NAMESPACE", ns.Name)
	c.Assert(err, check.IsNil)

	err = os.Setenv("POD_SERVICE_ACCOUNT", controllerSA)
	c.Assert(err, check.IsNil)

	err = os.Setenv(secureDefaultsEnvVar, "true")
	c.Assert(err, check.IsNil)

	err = createServiceAccount(s.cli, testSAName, s.namespace)
	c.Assert(err, check.IsNil)

	err = createServiceAccount(s.cli, dummyServiceAccount, s.namespace)
	c.Assert(err, check.IsNil)

	err = createServiceAccount(s.cli, controllerSA, s.namespace)
	c.Assert(err, check.IsNil)
}

func (s *PodSuiteSecureDefaults) TearDownSuite(c *check.C) {
	if s.namespace != "" {
		err := s.cli.CoreV1().Namespaces().Delete(context.TODO(), s.namespace, metav1.DeleteOptions{})
		c.Assert(err, check.IsNil)
	}

	err := os.Unsetenv(secureDefaultsEnvVar)
	c.Assert(err, check.IsNil)
}

func (s *PodSuiteSecureDefaults) TestPod(c *check.C) {
	// get controllers's namespace
	cns, err := GetControllerNamespace()
	c.Assert(err, check.IsNil)

	testcases := []*testcase{
		{
			description: "Create a pod with namespace, image, and node name, verify secure defaults are active",
			podOptions: &PodOptions{
				Namespace:     cns,
				GenerateName:  "test-",
				Image:         consts.LatestKanisterToolsImage,
				Command:       []string{"sh", "-c", "tail -f /dev/null"},
				RestartPolicy: corev1.RestartPolicyOnFailure,
			},
		},
		{
			description: "Create a pod with service account, verify secure defaults are active",
			podOptions: &PodOptions{
				Namespace:          s.namespace,
				GenerateName:       "test-",
				Image:              consts.LatestKanisterToolsImage,
				Command:            []string{"sh", "-c", "tail -f /dev/null"},
				ServiceAccountName: testSAName,
				RestartPolicy:      corev1.RestartPolicyAlways,
			},
		},
		{
			description: "Create a pod with podOverride options, Verify secure defaults are getting overridden",
			podOptions: &PodOptions{
				Namespace:          cns,
				GenerateName:       "test-",
				Image:              consts.LatestKanisterToolsImage,
				Command:            []string{"sh", "-c", "tail -f /dev/null"},
				ServiceAccountName: testSAName,
				RestartPolicy:      corev1.RestartPolicyNever,
				PodOverride: crv1alpha1.JSONMap{
					"serviceAccountName":           dummyServiceAccount,
					"automountServiceAccountToken": false,
				},
			},
		},
	}

	for _, t := range testcases {
		po := t.podOptions
		pod, err := CreatePod(context.Background(), s.cli, po)
		c.Assert(err, check.IsNil)

		c.Log("Test:", t.description)
		if po.PodOverride == nil {
			// secure Default is active.
			c.Assert(pod.Spec.ServiceAccountName, check.Equals, defaultServiceAccountName)
			c.Assert(pod.Spec.AutomountServiceAccountToken, check.NotNil)
			c.Assert(*pod.Spec.AutomountServiceAccountToken, check.Equals, false)
		} else {
			c.Assert(pod.Spec.ServiceAccountName, check.Equals, po.PodOverride["serviceAccountName"])
			c.Assert(pod.Spec.AutomountServiceAccountToken, check.NotNil)
			c.Assert(*pod.Spec.AutomountServiceAccountToken, check.Equals, po.PodOverride["automountServiceAccountToken"])
		}

		c.Assert(DeletePod(context.Background(), s.cli, pod), check.IsNil)
	}
}

func createServiceAccount(cli kubernetes.Interface, name string, ns string) error {
	sa := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}
	if _, err := cli.CoreV1().ServiceAccounts(ns).Create(context.TODO(), &sa, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}
