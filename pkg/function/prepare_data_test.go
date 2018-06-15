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

var _ = Suite(&PrepareDataSuite{})

type PrepareDataSuite struct {
	cli       kubernetes.Interface
	namespace string
}

func (s *PrepareDataSuite) SetUpSuite(c *C) {
	cli, err := kube.NewClient()
	c.Assert(err, IsNil)
	s.cli = cli

	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "preparedatatest-",
		},
	}
	cns, err := s.cli.Core().Namespaces().Create(ns)
	c.Assert(err, IsNil)
	s.namespace = cns.Name
}

func (s *PrepareDataSuite) TearDownSuite(c *C) {
	if s.namespace != "" {
		s.cli.Core().Namespaces().Delete(s.namespace, nil)
	}
}

func newPrepareDataBlueprint(pvc string) *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"test": {
				Kind: "StatefulSet",
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Name: "test",
						Func: "PrepareData",
						Args: map[string]interface{}{
							PrepareDataNamespaceArg: "{{ .StatefulSet.Namespace }}",
							PrepareDataImageArg:     "busybox",
							PrepareDataCommandArg: []string{
								"touch",
								"/mnt/data1/foo.txt",
							},
							PrepareDataVolumes: map[string]string{pvc: "/mnt/data1"},
						},
					},
					{
						Name: "test2",
						Func: "PrepareData",
						Args: map[string]interface{}{
							PrepareDataNamespaceArg: "{{ .StatefulSet.Namespace }}",
							PrepareDataImageArg:     "busybox",
							PrepareDataCommandArg: []string{
								"ls",
								"-l",
								"/mnt/data1/foo.txt",
							},
							PrepareDataVolumes: map[string]string{pvc: "/mnt/data1"},
						},
					},
				},
			},
		},
	}
}

func (s *PrepareDataSuite) TestPrepareData(c *C) {
	pvc := testutil.NewTestPVC()
	createdPVC, err := s.cli.CoreV1().PersistentVolumeClaims(s.namespace).Create(pvc)
	c.Assert(err, IsNil)

	ctx := context.Background()
	tp := param.TemplateParams{
		StatefulSet: &param.StatefulSetParams{
			Namespace: s.namespace,
		},
	}

	action := "test"
	bp := newPrepareDataBlueprint(createdPVC.Name)
	phases, err := kanister.GetPhases(*bp, action, tp)
	c.Assert(err, IsNil)
	for _, p := range phases {
		err := p.Exec(ctx, tp)
		c.Assert(err, IsNil)
	}
}
