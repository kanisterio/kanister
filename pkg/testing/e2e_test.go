// +build integration
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

package testing

import (
	"context"
	"time"

	"github.com/pkg/errors"
	. "gopkg.in/check.v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	crclient "github.com/kanisterio/kanister/pkg/client/clientset/versioned/typed/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/controller"
	"github.com/kanisterio/kanister/pkg/function"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/poll"
	"github.com/kanisterio/kanister/pkg/resource"
	"github.com/kanisterio/kanister/pkg/testutil"
)

type E2ESuite struct {
	cli       kubernetes.Interface
	crCli     crclient.CrV1alpha1Interface
	namespace string
	cancel    context.CancelFunc
}

var _ = Suite(&E2ESuite{})

func (s *E2ESuite) SetUpSuite(c *C) {
	// Instantiate Client SDKs
	cfg, err := kube.LoadConfig()
	c.Assert(err, IsNil)
	s.cli, err = kubernetes.NewForConfig(cfg)
	c.Assert(err, IsNil)
	s.crCli, err = crclient.NewForConfig(cfg)
	c.Assert(err, IsNil)

	// Create a new test namespace
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "e2e-test-",
		},
	}
	cns, err := s.cli.CoreV1().Namespaces().Create(ns)
	c.Assert(err, IsNil)
	s.namespace = cns.GetName()

	// Start the controller
	ctx := context.Background()
	ctx, s.cancel = context.WithCancel(ctx)
	err = resource.CreateCustomResources(ctx, cfg)
	c.Assert(err, IsNil)
	ctlr := controller.New(cfg)
	err = ctlr.StartWatch(ctx, s.namespace)
	c.Assert(err, IsNil)
}

func (s *E2ESuite) TearDownSuite(c *C) {
	if s.namespace != "" {
		s.cli.CoreV1().Namespaces().Delete(s.namespace, nil)
	}
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *E2ESuite) TestKubeExec(c *C) {
	ctx, can := context.WithTimeout(context.Background(), 60*time.Second)
	defer can()

	// Create a test Deployment
	d, err := s.cli.AppsV1().Deployments(s.namespace).Create(testutil.NewTestDeployment(1))
	c.Assert(err, IsNil)
	err = kube.WaitOnDeploymentReady(ctx, s.cli, s.namespace, d.GetName())
	c.Assert(err, IsNil)

	// Create a dummy Profile and secret
	sec := testutil.NewTestProfileSecret()
	sec, err = s.cli.CoreV1().Secrets(s.namespace).Create(sec)
	c.Assert(err, IsNil)

	p := testutil.NewTestProfile(s.namespace, sec.GetName())
	p, err = s.crCli.Profiles(s.namespace).Create(p)
	c.Assert(err, IsNil)

	// Create a simple Blueprint
	bp := &crv1alpha1.Blueprint{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-blueprint-",
		},
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"test": &crv1alpha1.BlueprintAction{
				Kind: "Deployment",
				Phases: []crv1alpha1.BlueprintPhase{
					crv1alpha1.BlueprintPhase{
						Func: function.KubeExecFuncName,
						Name: "test-kube-exec",
						Args: map[string]interface{}{
							"namespace": "{{ .Deployment.Namespace }}",
							"pod":       "{{ index .Deployment.Pods 0 }}",
							"container": "test-container",
							"command":   []string{"echo", "hello"},
						},
					},
				},
			},
		},
	}
	bp, err = s.crCli.Blueprints(s.namespace).Create(bp)
	c.Assert(err, IsNil)

	// Create an ActionSet
	as := &crv1alpha1.ActionSet{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-actionset-",
		},
		Spec: &crv1alpha1.ActionSetSpec{
			Actions: []crv1alpha1.ActionSpec{
				crv1alpha1.ActionSpec{
					Name: "test",
					Object: crv1alpha1.ObjectReference{
						Kind:      "Deployment",
						Name:      d.GetName(),
						Namespace: s.namespace,
					},
					Blueprint: bp.GetName(),
					Profile: &crv1alpha1.ObjectReference{
						Name:      p.GetName(),
						Namespace: s.namespace,
					},
				},
			},
		},
	}
	as, err = s.crCli.ActionSets(s.namespace).Create(as)
	c.Assert(err, IsNil)

	// Wait for the ActionSet to complete.
	err = poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		as, err = s.crCli.ActionSets(s.namespace).Get(as.GetName(), metav1.GetOptions{})
		switch {
		case err != nil, as.Status == nil:
			return false, err
		case as.Status.State == crv1alpha1.StateFailed:
			return true, errors.Errorf("Actionset failed: %#v", as.Status)
		case as.Status.State == crv1alpha1.StateComplete:
			return true, nil
		}
		return false, nil
	})
	c.Assert(err, IsNil)
}

func (s *E2ESuite) TestKubeTask(c *C) {
	ctx, can := context.WithTimeout(context.Background(), 30*time.Second)
	defer can()

	// Create a test Deployment
	d, err := s.cli.AppsV1().Deployments(s.namespace).Create(testutil.NewTestDeployment(1))
	c.Assert(err, IsNil)
	err = kube.WaitOnDeploymentReady(ctx, s.cli, s.namespace, d.GetName())
	c.Assert(err, IsNil)

	// Create a dummy Profile and secret
	sec := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-secret-",
		},
		StringData: map[string]string{
			"id":  "foo",
			"key": "bar",
		},
	}
	sec, err = s.cli.CoreV1().Secrets(s.namespace).Create(sec)
	c.Assert(err, IsNil)
	p := &crv1alpha1.Profile{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-profile-",
		},
		Location: crv1alpha1.Location{
			Type: crv1alpha1.LocationTypeS3Compliant,
		},
		Credential: crv1alpha1.Credential{
			Type: crv1alpha1.CredentialTypeKeyPair,
			KeyPair: &crv1alpha1.KeyPair{
				IDField:     "id",
				SecretField: "key",
				Secret: crv1alpha1.ObjectReference{
					Name:      sec.GetName(),
					Namespace: s.namespace,
				},
			},
		},
	}
	p, err = s.crCli.Profiles(s.namespace).Create(p)
	c.Assert(err, IsNil)

	// Create a simple Blueprint
	bp := &crv1alpha1.Blueprint{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-blueprint-",
		},
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"test": &crv1alpha1.BlueprintAction{
				Kind: "Deployment",
				Phases: []crv1alpha1.BlueprintPhase{
					crv1alpha1.BlueprintPhase{
						Func: function.KubeTaskFuncName,
						Name: "test-kube-task",
						Args: map[string]interface{}{
							"image":     "kanisterio/kanister-tools:0.22.0",
							"namespace": "{{ .Deployment.Namespace }}",
							"command":   []string{"echo", "default specs"},
							"podOverride": map[string]interface{}{
								"containers": []map[string]interface{}{
									{
										"name":            "container",
										"imagePullPolicy": "IfNotPresent",
									},
								},
								"dnsPolicy": "Default",
							},
						},
					},
				},
			},
		},
	}
	bp, err = s.crCli.Blueprints(s.namespace).Create(bp)
	c.Assert(err, IsNil)

	// Create an ActionSet
	as := &crv1alpha1.ActionSet{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-actionset-",
		},
		Spec: &crv1alpha1.ActionSetSpec{
			Actions: []crv1alpha1.ActionSpec{
				crv1alpha1.ActionSpec{
					Name: "test",
					Object: crv1alpha1.ObjectReference{
						Kind:      "Deployment",
						Name:      d.GetName(),
						Namespace: s.namespace,
					},
					Blueprint: bp.GetName(),
					Profile: &crv1alpha1.ObjectReference{
						Name:      p.GetName(),
						Namespace: s.namespace,
					},
					PodOverride: map[string]interface{}{
						"dnsPolicy": "ClusterFirst",
					},
				},
			},
		},
	}
	as, err = s.crCli.ActionSets(s.namespace).Create(as)
	c.Assert(err, IsNil)

	// Wait for the ActionSet to complete.
	err = poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		as, err = s.crCli.ActionSets(s.namespace).Get(as.GetName(), metav1.GetOptions{})
		switch {
		case err != nil, as.Status == nil:
			return false, err
		case as.Status.State == crv1alpha1.StateFailed:
			return true, errors.Errorf("Actionset failed: %#v", as.Status)
		case as.Status.State == crv1alpha1.StateComplete:
			return true, nil
		}
		return false, nil
	})
	c.Assert(err, IsNil)
}
