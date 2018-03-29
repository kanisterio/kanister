package controller

import (
	"context"
	"reflect"
	"testing"
	"time"

	. "gopkg.in/check.v1"
	"k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/client/clientset/versioned/scheme"
	crclientv1alpha1 "github.com/kanisterio/kanister/pkg/client/clientset/versioned/typed/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/eventer"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/resource"
	"github.com/kanisterio/kanister/pkg/testutil"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type ControllerSuite struct {
	crCli      crclientv1alpha1.CrV1alpha1Interface
	cli        kubernetes.Interface
	namespace  string
	cancel     func()
	ss         *v1beta1.StatefulSet
	deployment *v1beta1.Deployment
	confimap   *v1.ConfigMap
	recorder   record.EventRecorder
}

var _ = Suite(&ControllerSuite{})

func (s *ControllerSuite) SetUpSuite(c *C) {
	config, err := kube.LoadConfig()
	c.Assert(err, IsNil)
	cli, err := kubernetes.NewForConfig(config)
	c.Assert(err, IsNil)
	crCli, err := crclientv1alpha1.NewForConfig(config)
	c.Assert(err, IsNil)

	// Make sure the CRD's exist.
	resource.CreateCustomResources(context.Background(), config)

	s.cli = cli
	s.crCli = crCli

	s.recorder = eventer.NewEventRecorder(s.cli, "Controller Test")

	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kanistercontrollertest-",
		},
	}
	cns, err := s.cli.Core().Namespaces().Create(ns)
	c.Assert(err, IsNil)
	s.namespace = cns.Name

	ss := testutil.NewTestStatefulSet()
	ss, err = s.cli.AppsV1beta1().StatefulSets(s.namespace).Create(ss)
	c.Assert(err, IsNil)
	s.ss = ss

	d := testutil.NewTestDeployment()
	d, err = s.cli.AppsV1beta1().Deployments(s.namespace).Create(d)
	c.Assert(err, IsNil)
	s.deployment = d

	cm := testutil.NewTestConfigMap()
	cm, err = s.cli.CoreV1().ConfigMaps(s.namespace).Create(cm)
	c.Assert(err, IsNil)
	s.confimap = cm

}

func (s *ControllerSuite) TearDownSuite(c *C) {
	if s.namespace != "" {
		s.cli.Core().Namespaces().Delete(s.namespace, nil)
	}
}

func (s *ControllerSuite) SetUpTest(c *C) {
	config, err := kube.LoadConfig()
	c.Assert(err, IsNil)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	ctlr := New(config)
	err = ctlr.StartWatch(ctx, s.namespace)
	c.Assert(err, IsNil)
	s.cancel = cancel
}

func (s *ControllerSuite) TearDownTest(c *C) {
	s.cancel()
}

func (s *ControllerSuite) TestWatch(c *C) {
	// We give it a few seconds complete it's scan. This isn't required for the
	// test, but is a more realistic startup scenario.
	time.Sleep(5 * time.Second)
}

func (s *ControllerSuite) waitOnActionSetState(c *C, as *crv1alpha1.ActionSet, state crv1alpha1.State) error {
	return wait.Poll(100*time.Millisecond, 10*time.Second, func() (bool, error) {
		as, err := s.crCli.ActionSets(as.GetNamespace()).Get(as.GetName(), metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return as.Status != nil && as.Status.State == state, nil
	})
}

func (s *ControllerSuite) TestEmptyActionSetStatus(c *C) {
	as := &crv1alpha1.ActionSet{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "testactionset-",
		},
		Spec: &crv1alpha1.ActionSetSpec{
			Actions: []crv1alpha1.ActionSpec{},
		},
	}
	as, err := s.crCli.ActionSets(s.namespace).Create(as)
	c.Assert(err, IsNil)
	defer func() {
		err := s.crCli.ActionSets(s.namespace).Delete(as.GetName(), nil)
		c.Assert(err, IsNil)
	}()

	err = s.waitOnActionSetState(c, as, crv1alpha1.StateComplete)
	c.Assert(err, IsNil)
}

func (s *ControllerSuite) TestSynchronousFailure(c *C) {
	as := &crv1alpha1.ActionSet{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "testactionset-",
		},
		Spec: &crv1alpha1.ActionSetSpec{
			Actions: []crv1alpha1.ActionSpec{
				crv1alpha1.ActionSpec{
					Blueprint: "NONEXISTANT_BLUEPRINT",
				},
			},
		},
	}
	as, err := s.crCli.ActionSets(s.namespace).Create(as)
	c.Assert(err, IsNil)
	defer func() {
		err := s.crCli.ActionSets(s.namespace).Delete(as.GetName(), nil)
		c.Assert(err, IsNil)
	}()

	err = s.waitOnActionSetState(c, as, crv1alpha1.StateFailed)
	c.Assert(err, IsNil)
}

func (s *ControllerSuite) TestExecActionSet(c *C) {
	for _, pok := range []string{"StatefulSet", "Deployment"} {
		for _, tc := range []struct {
			funcNames []string
			args      [][]string
		}{
			{
				funcNames: []string{testutil.WaitFuncName},
			},
			{
				funcNames: []string{testutil.WaitFuncName, testutil.WaitFuncName},
			},
			{
				funcNames: []string{testutil.FailFuncName},
			},
			{
				funcNames: []string{testutil.WaitFuncName, testutil.FailFuncName},
			},
			{
				funcNames: []string{testutil.FailFuncName, testutil.WaitFuncName},
			},
			{
				funcNames: []string{testutil.ArgFuncName},
			},
			{
				funcNames: []string{testutil.ArgFuncName, testutil.FailFuncName},
			},
		} {
			var err error
			// Add a blueprint with a mocked kanister function.
			bp := testutil.NewTestBlueprint(pok, tc.funcNames...)
			bp = testutil.BlueprintWithConfigMap(bp)
			bp, err = s.crCli.Blueprints(s.namespace).Create(bp)
			c.Assert(err, IsNil)

			var n string
			switch pok {
			case "StatefulSet":
				n = s.ss.GetName()
			case "Deployment":
				n = s.deployment.GetName()
			default:
				c.FailNow()
			}

			// Add an actionset that references that blueprint.
			as := testutil.NewTestActionSet(s.namespace, bp.GetName(), pok, n, s.namespace)
			as = testutil.ActionSetWithConfigMap(as, s.confimap.GetName())
			as, err = s.crCli.ActionSets(s.namespace).Create(as)
			c.Assert(err, IsNil)

			err = s.waitOnActionSetState(c, as, crv1alpha1.StateRunning)
			c.Assert(err, IsNil)

			final := crv1alpha1.StateComplete
		Loop:
			for _, fn := range tc.funcNames {
				switch fn {
				case testutil.FailFuncName:
					final = crv1alpha1.StateFailed
					break Loop
				case testutil.WaitFuncName:
					testutil.ReleaseWaitFunc()
				case testutil.ArgFuncName:
					c.Assert(testutil.ArgFuncArgs(), DeepEquals, []string{"myValue"})
				}
			}

			err = s.waitOnActionSetState(c, as, final)
			c.Assert(err, IsNil)

			err = s.crCli.Blueprints(s.namespace).Delete(bp.GetName(), nil)
			c.Assert(err, IsNil)
			err = s.crCli.ActionSets(s.namespace).Delete(as.GetName(), nil)
			c.Assert(err, IsNil)
		}
	}
}

func (s *ControllerSuite) TestActionSetEventLogs(c *C) {
	c.Skip("This may not work in MiniKube")
	as := &crv1alpha1.ActionSet{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "testactionset-",
		},
		Spec: &crv1alpha1.ActionSetSpec{
			Actions: []crv1alpha1.ActionSpec{
				crv1alpha1.ActionSpec{
					Blueprint: "NONEXISTANT_BLUEPRINT",
				},
			},
		},
	}
	as, err := s.crCli.ActionSets(s.namespace).Create(as)
	c.Assert(err, IsNil)

	if as.Kind == "" {
		as.Kind = reflect.TypeOf(crv1alpha1.ActionSet{}).Name()
	}
	msg := "Unit testing event logs"
	s.recorder.Event(as, v1.EventTypeWarning, "Error", msg)

	events, err := s.cli.CoreV1().Events(as.Namespace).Search(scheme.Scheme, as)
	c.Assert(err, IsNil)
	c.Assert(events, NotNil)
	c.Assert(len(events.Items) > 0, Equals, true)
	c.Assert(events.Items[0].Message, Equals, msg)
}
