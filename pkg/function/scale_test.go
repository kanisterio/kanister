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

package function

import (
	"context"
	"fmt"

	osversioned "github.com/openshift/client-go/apps/clientset/versioned"
	. "gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	k8sscheme "k8s.io/client-go/kubernetes/scheme"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/client/clientset/versioned"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/resource"
	"github.com/kanisterio/kanister/pkg/testutil"
)

type ScaleSuite struct {
	cli       kubernetes.Interface
	crCli     versioned.Interface
	osCli     osversioned.Interface
	namespace string
}

var _ = Suite(&ScaleSuite{})

func (s *ScaleSuite) SetUpTest(c *C) {
	config, err := kube.LoadConfig()
	c.Assert(err, IsNil)
	cli, err := kubernetes.NewForConfig(config)
	c.Assert(err, IsNil)
	crCli, err := versioned.NewForConfig(config)
	c.Assert(err, IsNil)
	osCli, err := osversioned.NewForConfig(config)
	c.Assert(err, IsNil)

	s.cli = cli
	s.crCli = crCli
	s.osCli = osCli
	ctx := context.Background()
	err = resource.CreateCustomResources(context.Background(), config)
	c.Assert(err, IsNil)

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kanister-scale-test-",
		},
	}
	cns, err := s.cli.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	s.namespace = cns.Name

	sec := testutil.NewTestProfileSecret()
	sec, err = s.cli.CoreV1().Secrets(s.namespace).Create(ctx, sec, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	p := testutil.NewTestProfile(s.namespace, sec.GetName())
	_, err = crCli.CrV1alpha1().Profiles(s.namespace).Create(ctx, p, metav1.CreateOptions{})
	c.Assert(err, IsNil)
}

func (s *ScaleSuite) TearDownTest(c *C) {
	if s.namespace != "" {
		_ = s.cli.CoreV1().Namespaces().Delete(context.TODO(), s.namespace, metav1.DeleteOptions{})
	}
}

func newScaleBlueprint(kind string, scaleUpCount string) *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"echoHello": {
				Kind: kind,
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Name: "testScale",
						Func: KubeExecFuncName,
						Args: map[string]interface{}{
							KubeExecNamespaceArg:     fmt.Sprintf("{{ .%s.Namespace }}", kind),
							KubeExecPodNameArg:       fmt.Sprintf("{{ index .%s.Pods 1 }}", kind),
							KubeExecContainerNameArg: fmt.Sprintf("{{ index .%s.Containers 0 0 }}", kind),
							KubeExecCommandArg:       []string{"echo", "hello"},
						},
					},
				},
			},
			"scaleDown": {
				Kind: kind,
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Name: "testScale",
						Func: ScaleWorkloadFuncName,
						Args: map[string]interface{}{
							ScaleWorkloadReplicas: 0,
						},
					},
				},
			},
			"scaleUp": {
				Kind: kind,
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Name: "testScale",
						Func: ScaleWorkloadFuncName,
						Args: map[string]interface{}{
							ScaleWorkloadReplicas: scaleUpCount,
						},
					},
				},
			},
		},
	}
}

func (s *ScaleSuite) TestScaleDeployment(c *C) {
	ctx := context.Background()
	var originalReplicaCount int32 = 1
	d := testutil.NewTestDeployment(originalReplicaCount)
	d.Spec.Template.Spec.Containers[0].Lifecycle = &corev1.Lifecycle{
		PreStop: &corev1.LifecycleHandler{
			Exec: &corev1.ExecAction{
				Command: []string{"sleep", "30"},
			},
		},
	}

	d, err := s.cli.AppsV1().Deployments(s.namespace).Create(ctx, d, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	err = kube.WaitOnDeploymentReady(ctx, s.cli, d.GetNamespace(), d.GetName())
	c.Assert(err, IsNil)

	kind := "Deployment"
	as := crv1alpha1.ActionSpec{
		Object: crv1alpha1.ObjectReference{
			Kind:      kind,
			Namespace: s.namespace,
			Name:      d.GetName(),
		},
		Profile: &crv1alpha1.ObjectReference{
			Name:      testutil.TestProfileName,
			Namespace: s.namespace,
		},
	}
	var scaleUpToReplicas int32 = 2
	for _, action := range []string{"scaleUp", "echoHello", "scaleDown"} {
		tp, err := param.New(ctx, s.cli, fake.NewSimpleDynamicClient(k8sscheme.Scheme, d), s.crCli, s.osCli, as)
		c.Assert(err, IsNil)
		bp := newScaleBlueprint(kind, fmt.Sprintf("%d", scaleUpToReplicas))
		phases, err := kanister.GetPhases(*bp, action, kanister.DefaultVersion, *tp)
		c.Assert(err, IsNil)
		for _, p := range phases {
			out, err := p.Exec(context.Background(), *bp, action, *tp)
			c.Assert(err, IsNil)
			// at the start workload has `originalReplicaCount` replicas, the first phase that is going to get executed is
			// `scaleUp` which would change that count to 2, but the function would return the count that workload originally had
			// i.e., `originalReplicaCount`
			if action == "scaleUp" {
				c.Assert(out[outputArtifactOriginalReplicaCount], Equals, originalReplicaCount)
			}
			// `scaleDown` is going to change the replica count to 0 from 2. Because the workload already had 2 replicas
			//  (previous phase), so ouptut artifact from the function this time would be what the workload already had i.e., 2
			if action == "scaleDown" {
				c.Assert(out[outputArtifactOriginalReplicaCount], Equals, scaleUpToReplicas)
			}
		}
		ok, _, err := kube.DeploymentReady(ctx, s.cli, d.GetNamespace(), d.GetName())
		c.Assert(err, IsNil)
		c.Assert(ok, Equals, true)
	}

	pods, err := s.cli.CoreV1().Pods(s.namespace).List(ctx, metav1.ListOptions{})
	c.Assert(err, IsNil)
	c.Assert(pods.Items, HasLen, 0)
}

func (s *ScaleSuite) TestScaleStatefulSet(c *C) {
	ctx := context.Background()
	var originalReplicaCount int32 = 1
	ss := testutil.NewTestStatefulSet(originalReplicaCount)
	ss.Spec.Template.Spec.Containers[0].Lifecycle = &corev1.Lifecycle{
		PreStop: &corev1.LifecycleHandler{
			Exec: &corev1.ExecAction{
				Command: []string{"sleep", "30"},
			},
		},
	}
	ss, err := s.cli.AppsV1().StatefulSets(s.namespace).Create(ctx, ss, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	err = kube.WaitOnStatefulSetReady(ctx, s.cli, ss.GetNamespace(), ss.GetName())
	c.Assert(err, IsNil)

	kind := "StatefulSet"
	as := crv1alpha1.ActionSpec{
		Object: crv1alpha1.ObjectReference{
			Kind:      kind,
			Name:      ss.GetName(),
			Namespace: s.namespace,
		},
		Profile: &crv1alpha1.ObjectReference{
			Name:      testutil.TestProfileName,
			Namespace: s.namespace,
		},
	}

	var scaleUpToReplicas int32 = 2
	for _, action := range []string{"scaleUp", "echoHello", "scaleDown"} {
		tp, err := param.New(ctx, s.cli, fake.NewSimpleDynamicClient(k8sscheme.Scheme, ss), s.crCli, s.osCli, as)
		c.Assert(err, IsNil)
		bp := newScaleBlueprint(kind, fmt.Sprintf("%d", scaleUpToReplicas))
		phases, err := kanister.GetPhases(*bp, action, kanister.DefaultVersion, *tp)
		c.Assert(err, IsNil)
		for _, p := range phases {
			out, err := p.Exec(context.Background(), *bp, action, *tp)
			c.Assert(err, IsNil)
			// at the start workload has `originalReplicaCount` replicas, the first phase that is going to get executed is
			// `scaleUp` which would change that count to 2, but the function would return the count that workload originally had
			// i.e., `originalReplicaCount`
			if action == "scaleUp" {
				c.Assert(out[outputArtifactOriginalReplicaCount], Equals, originalReplicaCount)
			}
			// `scaleDown` is going to change the replica count to 0 from 2. Because the workload already had 2 replicas
			//  (previous phase), so ouptut artifact from the function this time would be what the workload already had i.e., 2
			if action == "scaleDown" {
				c.Assert(out[outputArtifactOriginalReplicaCount], Equals, scaleUpToReplicas)
			}
		}
		ok, _, err := kube.StatefulSetReady(ctx, s.cli, ss.GetNamespace(), ss.GetName())
		c.Assert(err, IsNil)
		c.Assert(ok, Equals, true)
	}

	_, err = s.cli.CoreV1().Pods(s.namespace).List(ctx, metav1.ListOptions{})
	c.Assert(err, IsNil)
}

func (s *ScaleSuite) TestGetArgs(c *C) {
	for _, tc := range []struct {
		tp               param.TemplateParams
		args             map[string]interface{}
		wantNamespace    string
		wantKind         string
		wantName         string
		wantReplicas     int32
		wantWaitForReady bool
		check            Checker
	}{
		{
			tp:    param.TemplateParams{},
			args:  map[string]interface{}{ScaleWorkloadReplicas: 2},
			check: NotNil,
		},
		{
			tp: param.TemplateParams{},
			args: map[string]interface{}{
				ScaleWorkloadReplicas:     "2",
				ScaleWorkloadNamespaceArg: "foo",
				ScaleWorkloadNameArg:      "app",
				ScaleWorkloadKindArg:      param.StatefulSetKind,
				ScaleWorkloadWaitArg:      false,
			},
			wantKind:         param.StatefulSetKind,
			wantName:         "app",
			wantNamespace:    "foo",
			wantReplicas:     int32(2),
			wantWaitForReady: false,
			check:            IsNil,
		},
		{
			tp: param.TemplateParams{
				StatefulSet: &param.StatefulSetParams{
					Name:      "app",
					Namespace: "foo",
				},
			},
			args: map[string]interface{}{
				ScaleWorkloadReplicas: 2,
			},
			wantKind:         param.StatefulSetKind,
			wantName:         "app",
			wantNamespace:    "foo",
			wantReplicas:     int32(2),
			wantWaitForReady: true,
			check:            IsNil,
		},
		{
			tp: param.TemplateParams{
				Deployment: &param.DeploymentParams{
					Name:      "app",
					Namespace: "foo",
				},
			},
			args: map[string]interface{}{
				ScaleWorkloadReplicas: int64(2),
			},
			wantKind:         param.DeploymentKind,
			wantName:         "app",
			wantNamespace:    "foo",
			wantReplicas:     int32(2),
			wantWaitForReady: true,
			check:            IsNil,
		},
		{
			tp: param.TemplateParams{
				StatefulSet: &param.StatefulSetParams{
					Name:      "app",
					Namespace: "foo",
				},
			},
			args: map[string]interface{}{
				ScaleWorkloadReplicas:     int32(2),
				ScaleWorkloadNamespaceArg: "notfoo",
				ScaleWorkloadNameArg:      "notapp",
				ScaleWorkloadKindArg:      param.DeploymentKind,
			},
			wantKind:         param.DeploymentKind,
			wantName:         "notapp",
			wantNamespace:    "notfoo",
			wantReplicas:     int32(2),
			wantWaitForReady: true,
			check:            IsNil,
		},
	} {
		s := scaleWorkloadFunc{}
		err := s.setArgs(tc.tp, tc.args)
		c.Assert(err, tc.check)
		if err != nil {
			continue
		}
		c.Assert(s.namespace, Equals, tc.wantNamespace)
		c.Assert(s.name, Equals, tc.wantName)
		c.Assert(s.kind, Equals, tc.wantKind)
		c.Assert(s.replicas, Equals, tc.wantReplicas)
		c.Assert(s.waitForReady, Equals, tc.wantWaitForReady)
	}
}
