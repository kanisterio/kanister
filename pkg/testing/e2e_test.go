//go:build integration
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
	"fmt"
	"log"
	"time"

	"github.com/pkg/errors"
	. "gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
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
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "e2e-test-",
		},
	}
	cns, err := s.cli.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	s.namespace = cns.GetName()

	// Start the controller
	ctx := context.Background()
	ctx, s.cancel = context.WithCancel(ctx)
	err = resource.CreateCustomResources(ctx, cfg)
	c.Assert(err, IsNil)
	ctlr := controller.New(cfg, nil)
	err = ctlr.StartWatch(ctx, s.namespace)
	c.Assert(err, IsNil)
}

func (s *E2ESuite) TearDownSuite(c *C) {
	if s.namespace != "" {
		err := s.cli.CoreV1().Namespaces().Delete(context.TODO(), s.namespace, metav1.DeleteOptions{})
		c.Assert(err, IsNil)
	}
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *E2ESuite) TestKubeExec(c *C) {
	ctx, can := context.WithTimeout(context.Background(), 60*time.Second)
	defer can()

	// Create a test Deployment
	d, err := s.cli.AppsV1().Deployments(s.namespace).Create(ctx, testutil.NewTestDeployment(1), metav1.CreateOptions{})
	c.Assert(err, IsNil)
	err = kube.WaitOnDeploymentReady(ctx, s.cli, s.namespace, d.GetName())
	c.Assert(err, IsNil)

	// Create test Profile and secret
	sec := testutil.NewTestProfileSecret()
	sec, err = s.cli.CoreV1().Secrets(s.namespace).Create(ctx, sec, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	p := testutil.NewTestProfile(s.namespace, sec.GetName())
	p, err = s.crCli.Profiles(s.namespace).Create(ctx, p, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	// Create a simple Blueprint
	bp := &crv1alpha1.Blueprint{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-blueprint-",
		},
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"test": {
				Kind: "Deployment",
				Phases: []crv1alpha1.BlueprintPhase{
					{
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
	bp, err = s.crCli.Blueprints(s.namespace).Create(ctx, bp, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	// Create an ActionSet
	as := &crv1alpha1.ActionSet{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-actionset-",
		},
		Spec: &crv1alpha1.ActionSetSpec{
			Actions: []crv1alpha1.ActionSpec{
				{
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
	as, err = s.crCli.ActionSets(s.namespace).Create(ctx, as, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	// Wait for the ActionSet to complete.
	err = poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		as, err = s.crCli.ActionSets(s.namespace).Get(ctx, as.GetName(), metav1.GetOptions{})
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
	log.Print("Completed E2E TestKubeExec")
}

func (s *E2ESuite) TestKubeTask(c *C) {
	ctx, can := context.WithTimeout(context.Background(), 30*time.Second)
	defer can()

	// Create a test Deployment
	d, err := s.cli.AppsV1().Deployments(s.namespace).Create(ctx, testutil.NewTestDeployment(1), metav1.CreateOptions{})
	c.Assert(err, IsNil)
	err = kube.WaitOnDeploymentReady(ctx, s.cli, s.namespace, d.GetName())
	c.Assert(err, IsNil)

	// Create test Profile and secret
	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-secret-",
		},
		StringData: map[string]string{
			"id":  "foo",
			"key": "bar",
		},
	}
	sec, err = s.cli.CoreV1().Secrets(s.namespace).Create(ctx, sec, metav1.CreateOptions{})
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
	p, err = s.crCli.Profiles(s.namespace).Create(ctx, p, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	// Create a simple Blueprint
	bp := &crv1alpha1.Blueprint{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-blueprint-",
		},
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"test": {
				Kind: "Deployment",
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Func: function.KubeTaskFuncName,
						Name: "test-kube-task",
						Args: map[string]interface{}{
							"image":     "ghcr.io/kanisterio/kanister-tools:0.110.0",
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
	bp, err = s.crCli.Blueprints(s.namespace).Create(ctx, bp, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	// Create an ActionSet
	as := &crv1alpha1.ActionSet{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-actionset-",
		},
		Spec: &crv1alpha1.ActionSetSpec{
			Actions: []crv1alpha1.ActionSpec{
				{
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
	as, err = s.crCli.ActionSets(s.namespace).Create(ctx, as, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	// Wait for the ActionSet to complete.
	err = poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		as, err = s.crCli.ActionSets(s.namespace).Get(ctx, as.GetName(), metav1.GetOptions{})
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
	log.Print("Completed E2E TestKubeTask")
}

func (s *E2ESuite) TestPodLabelsAndAnnotations(c *C) {
	bp, err := s.crCli.Blueprints(s.namespace).Create(context.Background(), blueprintWithPodFunctions(), metav1.CreateOptions{})
	c.Assert(err, IsNil)

	// 1. scenario where the labels/annotations are provided via actionset as well as blueprint
	asPodLabels := map[string]string{
		"asLabKeyOne": "asLabValOne",
	}
	asPodAnn := map[string]string{
		"asAnnKeyOne": "asAnnValOne",
	}
	as := backupActionsetWihtPodLabelsAndAnnotations(s.namespace, bp.Name, asPodAnn, asPodLabels)
	asCreated, err := s.crCli.ActionSets(s.namespace).Create(context.Background(), as, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	err = s.waitForFunctionPodReady()
	c.Assert(err, IsNil)

	ctx := context.Background()
	pods, err := s.cli.CoreV1().Pods("default").List(ctx, metav1.ListOptions{
		LabelSelector: "createdBy=kanister",
	})
	c.Assert(err, IsNil)

	err = verifyLabelsInFunctionPod(pods.Items[0].Labels, map[string]string{
		"asLabKeyOne": "asLabValOne",
		"bpLabKeyOne": "bpLabValueOne",
		"labKey":      "labValue",
	})
	c.Assert(err, IsNil)
	err = verifyAnnotationsInFunctionPod(pods.Items[0].Annotations, map[string]string{
		"asAnnKeyOne": "asAnnValOne",
		"bpAnnKeyOne": "bpAnnValueOne",
		"annKey":      "annValue",
	})
	c.Assert(err, IsNil)

	err = s.waitForActionSetComplete(asCreated.Name)
	c.Assert(err, IsNil)

	// 2. scenario where labels/annotations are provided via actionset as well blueprint
	// and same key is present at both places.
	asOne := backupActionsetWihtPodLabelsAndAnnotations(s.namespace, bp.Name, map[string]string{
		"asAnnKeyOne": "asAnnValOne",
		"annKey":      "asAnnValue", // this annotation is present in blueprint as well but with diff value (annValue)
	}, map[string]string{
		"asLabKeyOne": "asLabValOne",
		"labKey":      "asLabValue", // this label is present in blueprint as well but with diff value (labValue)
	})
	asCreatedOne, err := s.crCli.ActionSets(s.namespace).Create(context.Background(), asOne, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	err = s.waitForFunctionPodReady()
	c.Assert(err, IsNil)

	pods, err = s.cli.CoreV1().Pods("default").List(ctx, metav1.ListOptions{
		LabelSelector: "createdBy=kanister",
	})
	c.Assert(err, IsNil)
	err = verifyLabelsInFunctionPod(pods.Items[0].Labels, map[string]string{
		"asLabKeyOne": "asLabValOne",
		"bpLabKeyOne": "bpLabValueOne",
		"labKey":      "labValue",
	})
	c.Assert(err, IsNil)
	err = verifyAnnotationsInFunctionPod(pods.Items[0].Annotations, map[string]string{
		"asAnnKeyOne": "asAnnValOne",
		"bpAnnKeyOne": "bpAnnValueOne",
		"annKey":      "annValue",
	})
	c.Assert(err, IsNil)

	err = s.waitForActionSetComplete(asCreatedOne.Name)
	c.Assert(err, IsNil)

	// 3. scenario where labels/annotations are present at both places (actionset, blueprint) and no common key is present
	asTwo := backupActionsetWihtPodLabelsAndAnnotations(s.namespace, bp.Name, map[string]string{
		"asAnnKeyOne": "asAnnValOne",
		"asAnnKeyTwo": "asAnnValTwo",
	}, map[string]string{
		"asLabKeyOne": "asLabValOne",
		"asLabKeyTwo": "asLabValTwo",
	})
	asCreatedTwo, err := s.crCli.ActionSets(s.namespace).Create(context.Background(), asTwo, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	err = s.waitForFunctionPodReady()
	c.Assert(err, IsNil)

	pods, err = s.cli.CoreV1().Pods("default").List(ctx, metav1.ListOptions{
		LabelSelector: "createdBy=kanister",
	})
	c.Assert(err, IsNil)
	err = verifyLabelsInFunctionPod(pods.Items[0].Labels, map[string]string{
		"asLabKeyOne": "asLabValOne",
		"asLabKeyTwo": "asLabValTwo",
		"bpLabKeyOne": "bpLabValueOne",
		"labKey":      "labValue",
	})
	c.Assert(err, IsNil)
	err = verifyAnnotationsInFunctionPod(pods.Items[0].Annotations, map[string]string{
		"asAnnKeyOne": "asAnnValOne",
		"asAnnKeyTwo": "asAnnValTwo",
		"bpAnnKeyOne": "bpAnnValueOne",
		"annKey":      "annValue",
	})
	c.Assert(err, IsNil)

	err = s.waitForActionSetComplete(asCreatedTwo.Name)
	c.Assert(err, IsNil)

	// 4. scenario where labels/annotations are only provided via blueprint
	asThree := backupActionsetWihtPodLabelsAndAnnotations(s.namespace, bp.Name, nil, nil)
	asCreatedThree, err := s.crCli.ActionSets(s.namespace).Create(context.Background(), asThree, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	err = s.waitForFunctionPodReady()
	c.Assert(err, IsNil)

	pods, err = s.cli.CoreV1().Pods("default").List(ctx, metav1.ListOptions{
		LabelSelector: "createdBy=kanister",
	})
	c.Assert(err, IsNil)
	err = verifyLabelsInFunctionPod(pods.Items[0].Labels, map[string]string{
		"bpLabKeyOne": "bpLabValueOne",
		"labKey":      "labValue",
	})
	c.Assert(err, IsNil)
	err = verifyAnnotationsInFunctionPod(pods.Items[0].Annotations, map[string]string{
		"bpAnnKeyOne": "bpAnnValueOne",
		"annKey":      "annValue",
	})
	c.Assert(err, IsNil)

	err = s.waitForActionSetComplete(asCreatedThree.Name)
	c.Assert(err, IsNil)

	// 5. scenario where labels/annotations are only provided via actionset
	bpObj := blueprintWithPodFunctions()
	bpObj.Actions["backup"].Phases[0].Args["podLabels"] = map[string]string{}
	bpObj.Actions["backup"].Phases[0].Args["podAnnotations"] = map[string]string{}
	bp, err = s.crCli.Blueprints(s.namespace).Create(context.Background(), bpObj, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	asFour := backupActionsetWihtPodLabelsAndAnnotations(s.namespace, bp.Name, map[string]string{
		"asAnnKeyOne": "asAnnValOne",
		"asAnnKeyTwo": "asAnnValTwo",
	}, map[string]string{
		"asLabKeyOne": "asLabValOne",
	})
	asCreatedFour, err := s.crCli.ActionSets(s.namespace).Create(context.Background(), asFour, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	err = s.waitForFunctionPodReady()
	c.Assert(err, IsNil)

	pods, err = s.cli.CoreV1().Pods("default").List(ctx, metav1.ListOptions{
		LabelSelector: "createdBy=kanister",
	})
	c.Assert(err, IsNil)
	err = verifyLabelsInFunctionPod(pods.Items[0].Labels, map[string]string{
		"asLabKeyOne": "asLabValOne",
	})
	c.Assert(err, IsNil)
	err = verifyAnnotationsInFunctionPod(pods.Items[0].Annotations, map[string]string{
		"asAnnKeyOne": "asAnnValOne",
		"asAnnKeyTwo": "asAnnValTwo",
	})
	c.Assert(err, IsNil)

	err = s.waitForActionSetComplete(asCreatedFour.Name)
	c.Assert(err, IsNil)

	// test restore actionset
	bpObj = blueprintWithPodFunctions()
	bp, err = s.crCli.Blueprints(s.namespace).Create(context.Background(), bpObj, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	asRestore := restoreActionsetWithPodLabelsAndAnnotations(s.namespace, bp.Name, map[string]string{
		"asAnnKeyOne": "asAnnValOne",
		"asAnnKeyTwo": "asAnnValTwo",
	}, map[string]string{
		"asLabKeyOne": "asLabValOne",
	})
	asRestoreCreated, err := s.crCli.ActionSets(s.namespace).Create(context.Background(), asRestore, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	err = s.waitForFunctionPodReady()
	c.Assert(err, IsNil)

	pods, err = s.cli.CoreV1().Pods("default").List(ctx, metav1.ListOptions{
		LabelSelector: "createdBy=kanister",
	})
	c.Assert(err, IsNil)
	err = verifyLabelsInFunctionPod(pods.Items[0].Labels, map[string]string{
		"bpLabKeyOne": "bpLabValueOne",
		"labKey":      "labValue",
		"asLabKeyOne": "asLabValOne",
	})
	c.Assert(err, IsNil)
	err = verifyAnnotationsInFunctionPod(pods.Items[0].Annotations, map[string]string{
		"asAnnKeyOne": "asAnnValOne",
		"asAnnKeyTwo": "asAnnValTwo",
		"bpAnnKeyOne": "bpAnnValueOne",
		"annKey":      "annValue",
	})
	c.Assert(err, IsNil)

	err = s.waitForActionSetComplete(asRestoreCreated.Name)
	c.Assert(err, IsNil)

	log.Print("Completed E2E TestPodLabelsAndAnnotations")
}

func (s *E2ESuite) waitForActionSetComplete(asName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	return poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		as, err := s.crCli.ActionSets(s.namespace).Get(ctx, asName, metav1.GetOptions{})
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
}

// waitForFunctionPodReady waits for the pod created by a Kanister function. The pods
// that get created by Kanister function have the label `createdBy=kanister`, that's
// why we are checking that label in the pods to make sure one pod is created.
func (s *E2ESuite) waitForFunctionPodReady() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		pods, err := s.cli.CoreV1().Pods("default").List(ctx, metav1.ListOptions{
			LabelSelector: "createdBy=kanister",
		})
		if err != nil {
			return false, nil
		}

		if len(pods.Items) == 0 {
			// the Kanister function pod has not come up yet
			return false, nil
		}

		if len(pods.Items) > 1 {
			return false, errors.New("more than one kanister-job pod found")
		}

		return true, nil
	})
}

func verifyAnnotationsInFunctionPod(funcPodAnnotations, expectedAnnotations map[string]string) error {
	for k, v := range expectedAnnotations {
		val, ok := funcPodAnnotations[k]
		if !ok || v != val {
			return errors.New(fmt.Sprintf("Either key %s, is not found in pod annotations or, its values (%s and %s) don't match", k, v, val))
		}
	}
	return nil
}

func verifyLabelsInFunctionPod(funcPodLabels, expectedLabels map[string]string) error {
	for k, v := range expectedLabels {
		val, ok := funcPodLabels[k]
		if !ok || v != val {
			return errors.New(fmt.Sprintf("Either key %s, is not found in pod labels or, its values (%s and %s) don't match", k, v, val))
		}
	}
	return nil
}

func backupActionsetWihtPodLabelsAndAnnotations(testNS, bpName string, annotations, labels map[string]string) *crv1alpha1.ActionSet {
	return actionSetWithAction("backup", testNS, bpName, annotations, labels)
}

func actionSetWithAction(action, testNS, bpName string, annotations, labels map[string]string) *crv1alpha1.ActionSet {
	return &crv1alpha1.ActionSet{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "actionset-",
		},
		Spec: &crv1alpha1.ActionSetSpec{
			Actions: []crv1alpha1.ActionSpec{
				{
					Name: action,
					Object: crv1alpha1.ObjectReference{
						Kind:      "Namespace",
						Name:      testNS,
						Namespace: testNS,
					},
					Blueprint:      bpName,
					PodLabels:      labels,
					PodAnnotations: annotations,
				},
			},
		},
	}
}

func restoreActionsetWithPodLabelsAndAnnotations(testNS, bpName string, annotations, labels map[string]string) *crv1alpha1.ActionSet {
	return actionSetWithAction("restore", testNS, bpName, annotations, labels)
}

// blueprintWithPodFunctions returns a blueprint resource that has kanister functions
// that create a pod.
func blueprintWithPodFunctions() *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "blueprint-",
		},
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"backup": {
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Func: function.KubeTaskFuncName,
						Name: "backupphase-one",
						Args: map[string]interface{}{
							"image":     "ghcr.io/kanisterio/kanister-tools:0.110.0",
							"namespace": "default",
							"command":   []string{"sleep", "10"},
							"podLabels": map[string]interface{}{
								"bpLabKeyOne": "bpLabValueOne",
								"labKey":      "labValue",
							},
							"podAnnotations": map[string]interface{}{
								"bpAnnKeyOne": "bpAnnValueOne",
								"annKey":      "annValue",
							},
						},
					},
				},
			},
			"restore": {
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Func: function.KubeTaskFuncName,
						Name: "restorephase-one",
						Args: map[string]interface{}{
							"image":     "ghcr.io/kanisterio/kanister-tools:0.110.0",
							"namespace": "default",
							"command":   []string{"sleep", "10"},
							"podLabels": map[string]interface{}{
								"bpLabKeyOne": "bpLabValueOne",
								"labKey":      "labValue",
							},
							"podAnnotations": map[string]interface{}{
								"bpAnnKeyOne": "bpAnnValueOne",
								"annKey":      "annValue",
							},
						},
					},
				},
			},
		},
	}
}
