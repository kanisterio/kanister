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

type ScaleSuite struct {
	cli       kubernetes.Interface
	namespace string
}

var _ = Suite(&ScaleSuite{})

func (s *ScaleSuite) SetUpTest(c *C) {
	s.cli = kube.NewClient()
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kanister-scale-test-",
		},
	}
	cns, err := s.cli.Core().Namespaces().Create(ns)
	c.Assert(err, IsNil)
	s.namespace = cns.Name
}

func (s *ScaleSuite) TearDownTest(c *C) {
	if s.namespace != "" {
		s.cli.Core().Namespaces().Delete(s.namespace, nil)
	}
}

func newScaleBlueprint(kind string) *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"echoHello": &crv1alpha1.BlueprintAction{
				Kind: kind,
				Phases: []crv1alpha1.BlueprintPhase{
					crv1alpha1.BlueprintPhase{
						Name: "testScale",
						Func: "KubeExec",
						Args: []string{
							fmt.Sprintf("{{ .%s.Namespace }}", kind),
							fmt.Sprintf("{{ index .%s.Pods 1 }}", kind),
							fmt.Sprintf("{{ index .%s.Containers 0 0 }}", kind),
							"echo",
							"hello",
						},
					},
				},
			},
			"scaleDown": &crv1alpha1.BlueprintAction{
				Kind: kind,
				Phases: []crv1alpha1.BlueprintPhase{
					crv1alpha1.BlueprintPhase{
						Name: "testScale",
						Func: "Scale" + kind,
						Args: []string{
							fmt.Sprintf("{{ .%s.Namespace }}", kind),
							fmt.Sprintf("{{ .%s.Name }}", kind),
							"0",
						},
					},
				},
			},
			"scaleUp": &crv1alpha1.BlueprintAction{
				Kind: kind,
				Phases: []crv1alpha1.BlueprintPhase{
					crv1alpha1.BlueprintPhase{
						Name: "testScale",
						Func: "Scale" + kind,
						Args: []string{
							fmt.Sprintf("{{ .%s.Namespace }}", kind),
							fmt.Sprintf("{{ .%s.Name }}", kind),
							"2",
						},
					},
				},
			},
		},
	}
}

func (s *ScaleSuite) TestScaleDeployment(c *C) {
	ctx := context.Background()
	d := testutil.NewTestDeployment()
	d.Spec.Template.Spec.Containers[0].Lifecycle = &v1.Lifecycle{
		PreStop: &v1.Handler{
			Exec: &v1.ExecAction{
				Command: []string{"sleep", "30"},
			},
		},
	}

	d, err := s.cli.AppsV1beta1().Deployments(s.namespace).Create(d)
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
	}
	for _, action := range []string{"scaleUp", "echoHello", "scaleDown"} {
		tp, err := param.New(ctx, s.cli, nil, as)
		c.Assert(err, IsNil)

		phases, err := kanister.GetPhases(*newScaleBlueprint(kind), action, *tp)
		c.Assert(err, IsNil)
		for _, p := range phases {
			err := p.Exec(context.Background())
			c.Assert(err, IsNil)
		}
		ok, err := kube.DeploymentReady(ctx, s.cli, d.GetNamespace(), d.GetName())
		c.Assert(err, IsNil)
		c.Assert(ok, Equals, true)
	}

	pods, err := s.cli.CoreV1().Pods(s.namespace).List(metav1.ListOptions{})
	c.Assert(err, IsNil)
	c.Assert(pods.Items, HasLen, 0)
}

func (s *ScaleSuite) TestScaleStatefulSet(c *C) {
	ctx := context.Background()
	ss := testutil.NewTestStatefulSet()
	ss.Spec.Template.Spec.Containers[0].Lifecycle = &v1.Lifecycle{
		PreStop: &v1.Handler{
			Exec: &v1.ExecAction{
				Command: []string{"sleep", "30"},
			},
		},
	}
	ss, err := s.cli.AppsV1beta1().StatefulSets(s.namespace).Create(ss)
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
	}

	for _, action := range []string{"scaleUp", "echoHello", "scaleDown"} {
		tp, err := param.New(ctx, s.cli, nil, as)
		c.Assert(err, IsNil)

		phases, err := kanister.GetPhases(*newScaleBlueprint(kind), action, *tp)
		c.Assert(err, IsNil)
		for _, p := range phases {
			err := p.Exec(context.Background())
			c.Assert(err, IsNil)

		}
		ok, err := kube.StatefulSetReady(ctx, s.cli, ss.GetNamespace(), ss.GetName())
		c.Assert(err, IsNil)
		c.Assert(ok, Equals, true)
	}

	pods, err := s.cli.CoreV1().Pods(s.namespace).List(metav1.ListOptions{})
	c.Assert(err, IsNil)
	for _, pod := range pods.Items {
		for _, cs := range pod.Status.ContainerStatuses {
			c.Assert(cs.State.Terminated, NotNil)
		}
	}
}
