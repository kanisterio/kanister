package function

import (
	"context"
	"fmt"

	. "gopkg.in/check.v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/testutil"
)

type KubeExecAllTest struct {
	cli       kubernetes.Interface
	namespace string
}

var _ = Suite(&KubeExecAllTest{})

func (s *KubeExecAllTest) SetUpSuite(c *C) {
	s.cli = kube.NewClient()
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kubeexecall-",
		},
	}
	cns, err := s.cli.Core().Namespaces().Create(ns)
	c.Assert(err, IsNil)
	s.namespace = cns.Name
}

func (s *KubeExecAllTest) TearDownSuite(c *C) {
	if s.namespace != "" {
		s.cli.Core().Namespaces().Delete(s.namespace, nil)
	}
}

func newExecAllBlueprint(kind string) *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"echo": &crv1alpha1.BlueprintAction{
				Kind: kind,
				Phases: []crv1alpha1.BlueprintPhase{
					crv1alpha1.BlueprintPhase{
						Name: "echoSomething",
						Func: "KubeExecAll",
						Args: map[string]interface{}{
							KubeExecAllNamespaceArg:      fmt.Sprintf("{{ .%s.Namespace }}", kind),
							KubeExecAllPodsNameArg:       fmt.Sprintf("{{ range .%s.Pods }} {{.}}{{ end }}", kind),
							KubeExecAllContainersNameArg: fmt.Sprintf("{{ index .%s.Containers 0 0 }}", kind),
							KubeExecAllCommandArg: []string{
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

func (s *KubeExecAllTest) TestKubeExecAllDeployment(c *C) {
	ctx := context.Background()
	d := testutil.NewTestDeployment()
	d, err := s.cli.AppsV1beta1().Deployments(s.namespace).Create(d)
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
	}
	tp, err := param.New(ctx, s.cli, nil, as)
	c.Assert(err, IsNil)

	action := "echo"
	phases, err := kanister.GetPhases(*newExecAllBlueprint(kind), action, *tp)
	c.Assert(err, IsNil)
	for _, p := range phases {
		err = p.Exec(ctx, *tp)
		c.Assert(err, IsNil)
	}
}

func (s *KubeExecAllTest) TestKubeExecAllStatefulSet(c *C) {
	ctx := context.Background()
	ss := testutil.NewTestStatefulSet()
	ss, err := s.cli.AppsV1beta1().StatefulSets(s.namespace).Create(ss)
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
	}
	tp, err := param.New(ctx, s.cli, nil, as)
	c.Assert(err, IsNil)

	action := "echo"
	phases, err := kanister.GetPhases(*newExecAllBlueprint(kind), action, *tp)
	c.Assert(err, IsNil)
	for _, p := range phases {
		err = p.Exec(ctx, *tp)
		c.Assert(err, IsNil)
	}
}
