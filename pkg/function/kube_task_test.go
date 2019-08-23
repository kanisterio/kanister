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
	cns, err := s.cli.CoreV1().Namespaces().Create(ns)
	c.Assert(err, IsNil)
	s.namespace = cns.Name
	os.Setenv("POD_NAMESPACE", cns.Name)
	os.Setenv("POD_SERVICE_ACCOUNT", "default")

}

func (s *KubeTaskSuite) TearDownSuite(c *C) {
	if s.namespace != "" {
		s.cli.CoreV1().Namespaces().Delete(s.namespace, nil)
	}
}

func outputPhase(namespace string) crv1alpha1.BlueprintPhase {
	return crv1alpha1.BlueprintPhase{
		Name: "testOutput",
		Func: "KubeTask",
		Args: map[string]interface{}{
			KubeTaskNamespaceArg: namespace,
			KubeTaskImageArg:     "kanisterio/kanister-tools:0.20.0",
			KubeTaskCommandArg: []string{
				"sh",
				"-c",
				"kando output version 0.20.0",
			},
		},
	}
}

func sleepPhase(namespace string) crv1alpha1.BlueprintPhase {
	return crv1alpha1.BlueprintPhase{
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
	}
}

func tickPhase(namespace string) crv1alpha1.BlueprintPhase {
	return crv1alpha1.BlueprintPhase{
		Name: "testTick",
		Func: "KubeTask",
		Args: map[string]interface{}{
			KubeTaskNamespaceArg: namespace,
			KubeTaskImageArg:     "alpine:3.10",
			KubeTaskCommandArg: []string{
				"sh",
				"-c",
				`for i in $(seq 3); do echo Tick: "${i}"; sleep 1; done`,
			},
		},
	}
}

func newTaskBlueprint(phases ...crv1alpha1.BlueprintPhase) *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"test": {
				Kind:   "StatefulSet",
				Phases: phases,
			},
		},
	}
}

func (s *KubeTaskSuite) TestKubeTask(c *C) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	tp := param.TemplateParams{
		StatefulSet: &param.StatefulSetParams{
			Namespace: s.namespace,
		},
	}
	action := "test"
	for _, tc := range []struct {
		bp   *crv1alpha1.Blueprint
		outs []map[string]interface{}
	}{
		{
			bp: newTaskBlueprint(outputPhase(s.namespace), sleepPhase(s.namespace), tickPhase(s.namespace)),
			outs: []map[string]interface{}{
				map[string]interface{}{
					"version": "0.20.0",
				},
				map[string]interface{}{},
				map[string]interface{}{},
			},
		},
	} {

		phases, err := kanister.GetPhases(*tc.bp, action, tp)
		c.Assert(err, IsNil)
		c.Assert(phases, HasLen, len(tc.outs))
		for i, p := range phases {
			out, err := p.Exec(ctx, *tc.bp, action, tp)
			c.Assert(err, IsNil, Commentf("Phase %s failed", p.Name()))
			c.Assert(out, DeepEquals, tc.outs[i])
		}
	}
}
