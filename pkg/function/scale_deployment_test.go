package function

import (
	"context"

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

type ScaleDeploymentTest struct {
	cli       kubernetes.Interface
	namespace string
}

var _ = Suite(&ScaleDeploymentTest{})

func (s *ScaleDeploymentTest) SetUpSuite(c *C) {
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

func (s *ScaleDeploymentTest) TearDownSuite(c *C) {
	if s.namespace != "" {
		s.cli.Core().Namespaces().Delete(s.namespace, nil)
	}
}

func newScaleDeploymentBlueprint() *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"scaleDown": &crv1alpha1.BlueprintAction{
				Kind: "Deployment",
				Phases: []crv1alpha1.BlueprintPhase{
					crv1alpha1.BlueprintPhase{
						Name: "testScale",
						Func: "ScaleDeployment",
						Args: []string{
							"{{ .Deployment.Namespace }}",
							"{{ .Deployment.Name }}",
							"2",
						},
					},
				},
			},
			"scaleUp": &crv1alpha1.BlueprintAction{
				Kind: "Deployment",
				Phases: []crv1alpha1.BlueprintPhase{
					crv1alpha1.BlueprintPhase{
						Name: "testScale",
						Func: "ScaleDeployment",
						Args: []string{
							"{{ .Deployment.Namespace }}",
							"{{ .Deployment.Name }}",
							"0",
						},
					},
				},
			},
		},
	}
}

func (s *ScaleDeploymentTest) TestScaleDeployment(c *C) {
	ctx := context.Background()
	d := testutil.NewTestDeployment()
	d, err := s.cli.AppsV1beta1().Deployments(s.namespace).Create(d)
	c.Assert(err, IsNil)

	ok := kube.WaitOnDeploymentReady(ctx, s.cli, d)
	c.Assert(ok, Equals, true)

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

	for _, action := range []string{"scaleUp", "scaleDown"} {
		phases, err := kanister.GetPhases(*newScaleDeploymentBlueprint(), action, *tp)
		c.Assert(err, IsNil)
		for _, p := range phases {
			err := p.Exec(context.Background())
			c.Assert(err, IsNil)
		}
		ok, err := kube.DeploymentReady(ctx, s.cli, d.GetNamespace(), d.GetName())
		c.Assert(err, IsNil)
		c.Assert(ok, Equals, true)
	}
}
