package function

import (
	"context"
	"os"

	. "gopkg.in/check.v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
)

var _ = Suite(&KubeTaskSuite{})

type KubeTaskSuite struct {
	cli       kubernetes.Interface
	namespace string
}

func (s *KubeTaskSuite) SetUpSuite(c *C) {
	s.cli = kube.NewClient()

	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kanisterdeletetest-",
		},
	}
	cns, err := s.cli.Core().Namespaces().Create(ns)
	c.Assert(err, IsNil)
	s.namespace = cns.Name
	os.Setenv("POD_NAMESPACE", cns.Name)
	os.Setenv("POD_SERVICE_ACCOUNT", "default")

}

func (s *KubeTaskSuite) TearDownSuite(c *C) {
	if s.namespace != "" {
		s.cli.Core().Namespaces().Delete(s.namespace, nil)
	}
}

func newTaskBlueprint() *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"test": {
				Kind: "StatefulSet",
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Name: "test",
						Func: "KubeTask",
						Args: []string{
							"namespace",
							"busybox",
							"sleep",
							"2",
						},
					},
					{
						Name: "test2",
						Func: "KubeTask",
						Args: []string{
							"test-namespace",
							"ubuntu:latest",
							"sleep",
							"2",
						},
					},
				},
			},
		},
	}
}

func (s *KubeTaskSuite) TestKubeTask(c *C) {
	ctx := context.Background()
	tp := param.TemplateParams{
		StatefulSet: &param.StatefulSetParams{
			Namespace: s.namespace,
		},
	}

	action := "test"
	phases, err := kanister.GetPhases(*newTaskBlueprint(), action, tp)
	c.Assert(err, IsNil)
	for _, p := range phases {
		err := p.Exec(ctx)
		c.Assert(err, IsNil)
	}
}
