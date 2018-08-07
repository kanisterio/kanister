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
	"github.com/kanisterio/kanister/pkg/client/clientset/versioned"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/resource"
	"github.com/kanisterio/kanister/pkg/testutil"
)

type KubeExecAllTest struct {
	crCli     versioned.Interface
	cli       kubernetes.Interface
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

	// Make sure the CRD's exist.
	err = resource.CreateCustomResources(context.Background(), config)
	c.Assert(err, IsNil)

	s.cli = cli
	s.crCli = crCli

	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kubeexecall-",
		},
	}
	cns, err := s.cli.Core().Namespaces().Create(ns)
	c.Assert(err, IsNil)
	s.namespace = cns.Name

	sec := testutil.NewTestProfileSecret()
	sec, err = s.cli.Core().Secrets(s.namespace).Create(sec)
	c.Assert(err, IsNil)

	p := testutil.NewTestProfile(s.namespace, sec.GetName())
	_, err = s.crCli.CrV1alpha1().Profiles(s.namespace).Create(p)
	c.Assert(err, IsNil)
}

func (s *KubeExecAllTest) TearDownSuite(c *C) {
	if s.namespace != "" {
		s.cli.Core().Namespaces().Delete(s.namespace, nil)
	}
}

func newExecAllBlueprint(kind string, replicas int32) *crv1alpha1.Blueprint {
	pods := make([]string, replicas)
	for i := int32(0); i < replicas; i++ {
		pods[i] = fmt.Sprintf("{{ index .%s.Pods %d }}", kind, i)
	}
	containers := []string{fmt.Sprintf("{{ index .%s.Containers 0 0 }}", kind)}

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
							KubeExecAllPodsNameArg:       pods,
							KubeExecAllContainersNameArg: containers,
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
		Profile: &crv1alpha1.ObjectReference{
			Name:      testutil.TestProfileName,
			Namespace: s.namespace,
		},
	}
	tp, err := param.New(ctx, s.cli, s.crCli, as)
	c.Assert(err, IsNil)

	action := "echo"
	phases, err := kanister.GetPhases(*newExecAllBlueprint(kind, *d.Spec.Replicas), action, *tp)
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
		Profile: &crv1alpha1.ObjectReference{
			Name:      testutil.TestProfileName,
			Namespace: s.namespace,
		},
	}
	tp, err := param.New(ctx, s.cli, s.crCli, as)
	c.Assert(err, IsNil)

	action := "echo"
	phases, err := kanister.GetPhases(*newExecAllBlueprint(kind, *ss.Spec.Replicas), action, *tp)
	c.Assert(err, IsNil)
	for _, p := range phases {
		err = p.Exec(ctx, *tp)
		c.Assert(err, IsNil)
	}
}
