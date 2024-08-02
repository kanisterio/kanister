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
	"strings"

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

type KubeExecTest struct {
	cli       kubernetes.Interface
	crCli     versioned.Interface
	osCli     osversioned.Interface
	namespace string
}

var _ = Suite(&KubeExecTest{})

func (s *KubeExecTest) SetUpSuite(c *C) {
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
			GenerateName: "kanisterkubeexectest-",
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

func (s *KubeExecTest) TearDownSuite(c *C) {
	if s.namespace != "" {
		_ = s.cli.CoreV1().Namespaces().Delete(context.TODO(), s.namespace, metav1.DeleteOptions{})
	}
}

func newKubeExecBlueprint() *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"echo": {
				Kind: "StatefulSet",
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Name: "echoSomething",
						Func: KubeExecFuncName,
						Args: map[string]interface{}{
							KubeExecNamespaceArg:     "{{ .StatefulSet.Namespace }}",
							KubeExecPodNameArg:       "{{ index .StatefulSet.Pods 0 }}",
							KubeExecContainerNameArg: "{{ index .StatefulSet.Containers 0 0 }}",
							KubeExecCommandArg: []string{
								"echo",
								"hello",
								"world"},
						},
					},
				},
			},
		},
	}
}

const ssSpec = `
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: %s
spec:
  replicas: 1
  serviceName: fake-svc
  selector:
    matchLabels:
      app: fake-app
  template:
    metadata:
      labels:
        app: fake-app
    spec:
      containers:
        - name: test-container
          image: alpine:3.6
          command: ["tail"]
          args: ["-f", "/dev/null"]
`

func (s *KubeExecTest) TestKubeExec(c *C) {
	ctx := context.Background()
	name := strings.ToLower(c.TestName())
	name = strings.Replace(name, ".", "", 1)
	spec := fmt.Sprintf(ssSpec, name)
	ss, err := kube.CreateStatefulSet(ctx, s.cli, s.namespace, spec)
	c.Assert(err, IsNil)
	err = kube.WaitOnStatefulSetReady(ctx, s.cli, ss.GetNamespace(), ss.GetName())
	c.Assert(err, IsNil)

	kind := "Statefulset"
	// Run the delete action.
	as := crv1alpha1.ActionSpec{
		Object: crv1alpha1.ObjectReference{
			Kind:      kind,
			Name:      name,
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
	bp := newKubeExecBlueprint()
	phases, err := kanister.GetPhases(*bp, action, kanister.DefaultVersion, *tp)
	c.Assert(err, IsNil)
	for _, p := range phases {
		_, err = p.Exec(context.Background(), *bp, action, *tp)
		c.Assert(err, IsNil)
	}
}

func (s *KubeExecTest) TestParseLogAndCreateOutput(c *C) {
	for _, tc := range []struct {
		log        string
		expected   map[string]interface{}
		errChecker Checker
		outChecker Checker
	}{
		{"###Phase-output###: {\"key\":\"version\",\"value\":\"0.110.0\"}", map[string]interface{}{"version": "0.110.0"}, IsNil, NotNil},
		{"###Phase-output###: {\"key\":\"version\",\"value\":\"0.110.0\"}\n###Phase-output###: {\"key\":\"path\",\"value\":\"/backup/path\"}",
			map[string]interface{}{"version": "0.110.0", "path": "/backup/path"}, IsNil, NotNil},
		{"Random message ###Phase-output###: {\"key\":\"version\",\"value\":\"0.110.0\"}", map[string]interface{}{"version": "0.110.0"}, IsNil, NotNil},
		{"Random message with newline \n###Phase-output###: {\"key\":\"version\",\"value\":\"0.110.0\"}", map[string]interface{}{"version": "0.110.0"}, IsNil, NotNil},
		{"###Phase-output###: Invalid message", nil, NotNil, IsNil},
		{"Random message", nil, IsNil, IsNil},
	} {
		out, err := parseLogAndCreateOutput(tc.log)
		c.Check(err, tc.errChecker)
		c.Check(out, tc.outChecker)
		if out != nil {
			c.Check(out, DeepEquals, tc.expected)
		}
	}
}
