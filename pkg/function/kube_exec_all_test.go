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

type KubeExecAllTest struct {
	crCli     versioned.Interface
	cli       kubernetes.Interface
	osCli     osversioned.Interface
	namespace string
}

var _ = Suite(&KubeExecAllTest{})

func (s *KubeExecAllTest) SetUpSuite(c *C) {
	config, err := kube.LoadConfig()
	c.Assert(err, IsNil)
	cli, err := kubernetes.NewForConfig(config)
	c.Assert(err, IsNil)
	crCli, err := versioned.NewForConfig(config)
	c.Assert(err, IsNil)
	osCli, err := osversioned.NewForConfig(config)
	c.Assert(err, IsNil)

	// Make sure the CRD's exist.
	err = resource.CreateCustomResources(context.Background(), config)
	c.Assert(err, IsNil)

	s.cli = cli
	s.crCli = crCli
	s.osCli = osCli

	ctx := context.Background()
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kubeexecall-",
		},
	}
	cns, err := s.cli.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	s.namespace = cns.Name

	sec := testutil.NewTestProfileSecret()
	sec, err = s.cli.CoreV1().Secrets(s.namespace).Create(ctx, sec, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	p := testutil.NewTestProfile(s.namespace, sec.GetName())
	_, err = s.crCli.CrV1alpha1().Profiles(s.namespace).Create(ctx, p, metav1.CreateOptions{})
	c.Assert(err, IsNil)
}

func (s *KubeExecAllTest) TearDownSuite(c *C) {
	if s.namespace != "" {
		_ = s.cli.CoreV1().Namespaces().Delete(context.TODO(), s.namespace, metav1.DeleteOptions{})
	}
}

func newExecAllBlueprint(kind string) *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"echo": {
				Kind: kind,
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Name: "echoSomething",
						Func: KubeExecAllFuncName,
						Args: map[string]interface{}{
							KubeExecAllNamespaceArg:      fmt.Sprintf("{{ .%s.Namespace }}", kind),
							KubeExecAllPodsNameArg:       fmt.Sprintf("{{ range .%s.Pods }} {{.}}{{ end }}", kind),
							KubeExecAllContainersNameArg: fmt.Sprintf("{{ index .%s.Containers 0 0 }}", kind),
							KubeExecAllCommandArg:        []string{"echo", "hello", "world"},
						},
					},
				},
			},
		},
	}
}

func (s *KubeExecAllTest) TestKubeExecAllDeployment(c *C) {
	ctx := context.Background()
	d := testutil.NewTestDeployment(1)
	d, err := s.cli.AppsV1().Deployments(s.namespace).Create(context.TODO(), d, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	err = kube.WaitOnDeploymentReady(ctx, s.cli, d.Namespace, d.Name)
	c.Assert(err, IsNil)

	kind := "Deployment"
	as := crv1alpha1.ActionSpec{
		Object: crv1alpha1.ObjectReference{
			Kind:      kind,
			Name:      d.GetName(),
			Namespace: s.namespace,
		},
		Profile: &crv1alpha1.ObjectReference{
			Name:      testutil.TestProfileName,
			Namespace: s.namespace,
		},
	}
	tp, err := param.New(ctx, s.cli, fake.NewSimpleDynamicClient(k8sscheme.Scheme, d), s.crCli, s.osCli, as)
	c.Assert(err, IsNil)

	action := "echo"
	bp := newExecAllBlueprint(kind)
	phases, err := kanister.GetPhases(*bp, action, kanister.DefaultVersion, *tp)
	c.Assert(err, IsNil)
	for _, p := range phases {
		_, err = p.Exec(ctx, *bp, action, *tp)
		c.Assert(err, IsNil)
	}
}

func (s *KubeExecAllTest) TestKubeExecAllStatefulSet(c *C) {
	ctx := context.Background()
	ss := testutil.NewTestStatefulSet(1)
	ss, err := s.cli.AppsV1().StatefulSets(s.namespace).Create(context.TODO(), ss, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	err = kube.WaitOnStatefulSetReady(ctx, s.cli, ss.Namespace, ss.Name)
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
	tp, err := param.New(ctx, s.cli, fake.NewSimpleDynamicClient(k8sscheme.Scheme, ss), s.crCli, s.osCli, as)
	c.Assert(err, IsNil)

	action := "echo"
	bp := newExecAllBlueprint(kind)
	phases, err := kanister.GetPhases(*bp, action, kanister.DefaultVersion, *tp)
	c.Assert(err, IsNil)
	for _, p := range phases {
		_, err = p.Exec(ctx, *bp, action, *tp)
		c.Assert(err, IsNil)
	}
}
