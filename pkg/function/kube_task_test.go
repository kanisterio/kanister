package function

import (
	"context"
	"os"
	"time"

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
	cli, err := kube.NewClient()
	c.Assert(err, IsNil)
	s.cli = cli

	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kanisterkubetasktest-",
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

func newTaskBlueprint(namespace string) *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"test": {
				Kind: "StatefulSet",
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Name: "testOutput",
						Func: "KubeTask",
						Args: map[string]interface{}{
							KubeTaskNamespaceArg: namespace,
							KubeTaskImageArg:     "kanisterio/kanister-tools:0.16.0",
							KubeTaskCommandArg: []string{
								"sh",
								"-c",
								"kando output version 0.16.0",
							},
						},
					},
					{
						Name: "testSleep",
						Func: "KubeTask",
						Args: map[string]interface{}{
							KubeTaskNamespaceArg: namespace,
							KubeTaskImageArg:     "ubuntu:latest",
							KubeTaskCommandArg: []string{
								"sleep",
								"2",
							},
						},
					},
				},
			},
		},
	}
}

func (s *KubeTaskSuite) TestKubeTask(c *C) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	tp := param.TemplateParams{
		StatefulSet: &param.StatefulSetParams{
			Namespace: s.namespace,
		},
	}

	action := "test"
	bp := newTaskBlueprint(s.namespace)
	phases, err := kanister.GetPhases(*bp, action, tp)
	c.Assert(err, IsNil)
	for _, p := range phases {
		out, err := p.Exec(ctx, *bp, action, tp)
		c.Assert(err, IsNil, Commentf("Phase %s failed", p.Name()))
		if out != nil {
			c.Assert(out["version"], NotNil)
		}
	}
}
