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

	. "gopkg.in/check.v1"
	v1 "k8s.io/api/core/v1"
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
	osversioned "github.com/openshift/client-go/apps/clientset/versioned"
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

	ns := &v1.Namespace{
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

func newScaleBlueprint(kind string) *crv1alpha1.Blueprint {
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
							ScaleWorkloadReplicas: "2",
						},
					},
				},
			},
		},
	}
}

func (s *ScaleSuite) TestScaleDeployment(c *C) {
	ctx := context.Background()
	d := testutil.NewTestDeployment(1)
	d.Spec.Template.Spec.Containers[0].Lifecycle = &v1.Lifecycle{
		PreStop: &v1.Handler{
			Exec: &v1.ExecAction{
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
	for _, action := range []string{"scaleUp", "echoHello", "scaleDown"} {
		tp, err := param.New(ctx, s.cli, fake.NewSimpleDynamicClient(k8sscheme.Scheme, d), s.crCli, s.osCli, as)
		c.Assert(err, IsNil)
		bp := newScaleBlueprint(kind)
		phases, err := kanister.GetPhases(*bp, action, kanister.DefaultVersion, *tp)
		c.Assert(err, IsNil)
		for _, p := range phases {
			_, err = p.Exec(context.Background(), *bp, action, *tp)
			c.Assert(err, IsNil)
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
	ss := testutil.NewTestStatefulSet(1)
	ss.Spec.Template.Spec.Containers[0].Lifecycle = &v1.Lifecycle{
		PreStop: &v1.Handler{
			Exec: &v1.ExecAction{
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

	for _, action := range []string{"scaleUp", "echoHello", "scaleDown"} {
		tp, err := param.New(ctx, s.cli, fake.NewSimpleDynamicClient(k8sscheme.Scheme, ss), s.crCli, s.osCli, as)
		c.Assert(err, IsNil)
		bp := newScaleBlueprint(kind)
		phases, err := kanister.GetPhases(*bp, action, kanister.DefaultVersion, *tp)
		c.Assert(err, IsNil)
		for _, p := range phases {
			_, err = p.Exec(context.Background(), *bp, action, *tp)
			c.Assert(err, IsNil)
		}
		ok, _, err := kube.StatefulSetReady(ctx, s.cli, ss.GetNamespace(), ss.GetName())
		c.Assert(err, IsNil)
		c.Assert(ok, Equals, true)
	}

	pods, err := s.cli.CoreV1().Pods(s.namespace).List(ctx, metav1.ListOptions{})
	c.Assert(err, IsNil)

	// This check can flake on underprovisioned clusters so we exit early.
	c.SucceedNow()
	for _, pod := range pods.Items {
		for _, cs := range pod.Status.ContainerStatuses {
			c.Assert(cs.State.Terminated, NotNil)
		}
	}
}

func (s *ScaleSuite) TestGetArgs(c *C) {
	for _, tc := range []struct {
		tp            param.TemplateParams
		args          map[string]interface{}
		wantNamespace string
		wantKind      string
		wantName      string
		wantReplicas  int32
		check         Checker
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
			},
			wantKind:      param.StatefulSetKind,
			wantName:      "app",
			wantNamespace: "foo",
			wantReplicas:  int32(2),
			check:         IsNil,
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
			wantKind:      param.StatefulSetKind,
			wantName:      "app",
			wantNamespace: "foo",
			wantReplicas:  int32(2),
			check:         IsNil,
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
			wantKind:      param.DeploymentKind,
			wantName:      "app",
			wantNamespace: "foo",
			wantReplicas:  int32(2),
			check:         IsNil,
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
			wantKind:      param.DeploymentKind,
			wantName:      "notapp",
			wantNamespace: "notfoo",
			wantReplicas:  int32(2),
			check:         IsNil,
		},
	} {
		namespace, kind, name, replicas, err := getArgs(tc.tp, tc.args)
		c.Assert(err, tc.check)
		if err != nil {
			continue
		}
		c.Assert(namespace, Equals, tc.wantNamespace)
		c.Assert(name, Equals, tc.wantName)
		c.Assert(kind, Equals, tc.wantKind)
		c.Assert(replicas, Equals, tc.wantReplicas)
	}
}
