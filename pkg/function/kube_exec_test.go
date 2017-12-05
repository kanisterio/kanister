package function

import (
	"context"
	"fmt"
	"strings"

	. "gopkg.in/check.v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
)

type KubeExecTest struct {
	cli       kubernetes.Interface
	namespace string
}

var _ = Suite(&KubeExecTest{})

func (s *KubeExecTest) SetUpSuite(c *C) {
	s.cli = kube.NewClient()
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kanisterkubeexectest-",
		},
	}
	cns, err := s.cli.Core().Namespaces().Create(ns)
	c.Assert(err, IsNil)
	s.namespace = cns.Name
}

func (s *KubeExecTest) TearDownSuite(c *C) {
	if s.namespace != "" {
		s.cli.Core().Namespaces().Delete(s.namespace, nil)
	}
}

func newKubeExecBlueprint() *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"echo": &crv1alpha1.BlueprintAction{
				Kind: "StatefulSet",
				Phases: []crv1alpha1.BlueprintPhase{
					crv1alpha1.BlueprintPhase{
						Name: "echoSomething",
						Func: "KubeExec",
						Args: []string{
							"{{ .StatefulSet.Namespace }}",
							"{{ index .StatefulSet.Pods 0 }}",
							"{{ index .StatefulSet.Containers 0 0 }}",
							"echo",
							"hello",
							"world",
						},
					},
				},
			},
		},
	}
}

const ssSpec = `
apiVersion: apps/v1beta1
kind: StatefulSet
metadata:
  name: %s
spec:
  replicas: 1
  serviceName: fake-svc
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
	ok := kube.WaitOnStatefulSetReady(ctx, s.cli, ss)
	c.Assert(ok, Equals, true)

	kind := "Statefulset"
	// Run the delete action.
	as := crv1alpha1.ActionSpec{
		Object: crv1alpha1.ObjectReference{
			Kind:      kind,
			Name:      name,
			Namespace: s.namespace,
		},
	}
	tp, err := param.New(ctx, s.cli, as)
	c.Assert(err, IsNil)

	action := "echo"
	phases, err := kanister.GetPhases(*newKubeExecBlueprint(), action, *tp)
	c.Assert(err, IsNil)
	for _, p := range phases {
		err := p.Exec(context.Background())
		c.Assert(err, IsNil)
	}
}
