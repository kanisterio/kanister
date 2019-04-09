package controller

import (
	"context"
	"fmt"
	"testing"
	"time"

	. "gopkg.in/check.v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/client/clientset/versioned/scheme"
	crclientv1alpha1 "github.com/kanisterio/kanister/pkg/client/clientset/versioned/typed/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/eventer"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/poll"
	"github.com/kanisterio/kanister/pkg/resource"
	"github.com/kanisterio/kanister/pkg/testutil"
	"github.com/pkg/errors"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type ControllerSuite struct {
	crCli      crclientv1alpha1.CrV1alpha1Interface
	cli        kubernetes.Interface
	namespace  string
	cancel     func()
	ss         *appsv1.StatefulSet
	deployment *appsv1.Deployment
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

	sec := testutil.NewTestProfileSecret()
	sec, err = s.cli.Core().Secrets(s.namespace).Create(sec)
	c.Assert(err, IsNil)

	p := testutil.NewTestProfile(s.namespace, sec.GetName())
	_, err = s.crCli.Profiles(s.namespace).Create(p)
	c.Assert(err, IsNil)

	ss := testutil.NewTestStatefulSet()
	ss, err = s.cli.AppsV1().StatefulSets(s.namespace).Create(ss)
	c.Assert(err, IsNil)
	s.ss = ss

	d := testutil.NewTestDeployment()
	d, err = s.cli.AppsV1().Deployments(s.namespace).Create(d)
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := poll.Wait(ctx, func(context.Context) (bool, error) {
		as, err := s.crCli.ActionSets(as.GetNamespace()).Get(as.GetName(), metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if as.Status == nil {
			return false, nil
		}
		if as.Status.State == state {
			return true, nil
		}
		// These are non-terminal states.
		if as.Status.State == crv1alpha1.StatePending || as.Status.State == crv1alpha1.StateRunning {
			return false, nil
		}
		return false, errors.New(fmt.Sprintf("Unexpected state: %s", as.Status.State))

	})
	if err == nil {
		return nil
	}
	return errors.Wrapf(err, "State '%s' never reached", state)
}

func newBPWithOutputArtifact() *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-blueprint-",
		},
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"myAction": &crv1alpha1.BlueprintAction{
				OutputArtifacts: map[string]crv1alpha1.Artifact{
					"myArt": crv1alpha1.Artifact{
						KeyValue: map[string]string{
							"key": "{{ .Phases.myPhase0.Output.key }}",
						},
					},
				},
				Kind: "Deployment",
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Name: "myPhase0",
						Func: testutil.OutputFuncName,
					},
				},
			},
		},
	}
}

func newBPWithFakeOutputArtifact() *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-blueprint-",
		},
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"myAction": &crv1alpha1.BlueprintAction{
				OutputArtifacts: map[string]crv1alpha1.Artifact{
					"myArt": crv1alpha1.Artifact{
						KeyValue: map[string]string{
							"key": "{{ .Phases.myPhase0.Output.myKey }}",
						},
					},
				},
				Kind: "Deployment",
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Name: "myPhase0",
						Func: testutil.OutputFuncName,
					},
				},
			},
		},
	}
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
					Object: crv1alpha1.ObjectReference{
						Name: "foo",
						Kind: param.NamespaceKind,
					},
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
			name      string
		}{
			{
				funcNames: []string{testutil.WaitFuncName},
				name:      "WaitFunc",
			},
			{
				funcNames: []string{testutil.WaitFuncName, testutil.WaitFuncName},
				name:      "WaitWait",
			},
			{
				funcNames: []string{testutil.FailFuncName},
				name:      "FailFunc",
			},
			{
				funcNames: []string{testutil.WaitFuncName, testutil.FailFuncName},
				name:      "WaitFail",
			},
			{
				funcNames: []string{testutil.FailFuncName, testutil.WaitFuncName},
				name:      "FailWait",
			},
			{
				funcNames: []string{testutil.ArgFuncName},
				name:      "ArgFunc",
			},
			{
				funcNames: []string{testutil.ArgFuncName, testutil.FailFuncName},
				name:      "ArgFail",
			},
			{
				funcNames: []string{testutil.OutputFuncName},
				name:      "OutputFunc",
			},
			{
				funcNames: []string{testutil.CancelFuncName},
				name:      "CancelFunc",
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
			c.Assert(err, IsNil, Commentf("Failed case: %s", tc.name))

			err = s.waitOnActionSetState(c, as, crv1alpha1.StateRunning)
			c.Assert(err, IsNil, Commentf("Failed case: %s", tc.name))

			final := crv1alpha1.StateComplete
			cancel := false
		Loop:
			for _, fn := range tc.funcNames {
				switch fn {
				case testutil.FailFuncName:
					final = crv1alpha1.StateFailed
					c.Assert(testutil.FailFuncError().Error(), DeepEquals, "Kanister function failed", Commentf("Failed case: %s", tc.name))
					break Loop
				case testutil.WaitFuncName:
					testutil.ReleaseWaitFunc()
				case testutil.ArgFuncName:
					c.Assert(testutil.ArgFuncArgs(), DeepEquals, map[string]interface{}{"key": "myValue"}, Commentf("Failed case: %s", tc.name))
				case testutil.OutputFuncName:
					c.Assert(testutil.OutputFuncOut(), DeepEquals, map[string]interface{}{"key": "myValue"}, Commentf("Failed case: %s", tc.name))
				case testutil.CancelFuncName:
					err = s.crCli.ActionSets(s.namespace).Delete(as.GetName(), nil)
					c.Assert(err, IsNil)
					c.Assert(testutil.CancelFuncOut().Error(), DeepEquals, "context canceled")
					cancel = true
				}
			}

			if !cancel {
				err = s.waitOnActionSetState(c, as, final)
				c.Assert(err, IsNil, Commentf("Failed case: %s", tc.name))
			}
			err = s.crCli.Blueprints(s.namespace).Delete(bp.GetName(), nil)
			c.Assert(err, IsNil)
			err = s.crCli.ActionSets(s.namespace).Delete(as.GetName(), nil)
			if !cancel {
				c.Assert(err, IsNil)
			} else {
				c.Assert(err, NotNil)
			}
		}
	}
}

func (s *ControllerSuite) TestRuntimeObjEventLogs(c *C) {
	c.Skip("This may not work in MiniKube")
	// Create ActionSet
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
	msg := "Unit testing event logs"
	reason := "Test Logs"

	//Create nil ActionSet
	var nilAs = (*crv1alpha1.ActionSet)(nil)

	// Create Blueprint
	bp := testutil.NewTestBlueprint("StatefulSet", testutil.WaitFuncName)
	bp, err = s.crCli.Blueprints(s.namespace).Create(bp)
	c.Assert(err, IsNil)

	//Test the logAndErrorEvent function
	config, err := kube.LoadConfig()
	c.Assert(err, IsNil)
	ctlr := New(config)
	ctlr.logAndErrorEvent(msg, reason, errors.New("Testing Event Logs"), as, nilAs, bp)

	// Test ActionSet error event logging
	events, err := s.cli.CoreV1().Events(as.Namespace).Search(scheme.Scheme, as)
	c.Assert(err, IsNil)
	c.Assert(events, NotNil)
	c.Assert(len(events.Items) > 0, Equals, true)
	c.Assert(events.Items[0].Message, Equals, msg)

	//Testing nil ActionSet error event logging
	events, err = s.cli.CoreV1().Events(as.Namespace).Search(scheme.Scheme, nilAs)
	c.Assert(err, NotNil)
	c.Assert(len(events.Items), Equals, 0)

	//Testing Blueprint error event logging
	events, err = s.cli.CoreV1().Events(bp.Namespace).Search(scheme.Scheme, bp)
	c.Assert(err, IsNil)
	c.Assert(events, NotNil)
	c.Assert(len(events.Items) > 0, Equals, true)
	c.Assert(events.Items[0].Message, Equals, msg)

	//Testing empty Blueprint
	testbp := &crv1alpha1.Blueprint{}
	ctlr.logAndErrorEvent(msg, reason, errors.New("Testing Event Logs"), testbp)
	events, err = s.cli.CoreV1().Events(bp.Namespace).Search(scheme.Scheme, testbp)
	c.Assert(err, NotNil)
	c.Assert(len(events.Items), Equals, 0)
}

func (s *ControllerSuite) TestPhaseOutputAsArtifact(c *C) {
	// Create a blueprint that uses func output as artifact
	bp := newBPWithOutputArtifact()
	bp = testutil.BlueprintWithConfigMap(bp)
	bp, err := s.crCli.Blueprints(s.namespace).Create(bp)
	c.Assert(err, IsNil)

	// Add an actionset that references that blueprint.
	as := testutil.NewTestActionSet(s.namespace, bp.GetName(), "Deployment", s.deployment.GetName(), s.namespace)
	as = testutil.ActionSetWithConfigMap(as, s.confimap.GetName())
	as, err = s.crCli.ActionSets(s.namespace).Create(as)
	c.Assert(err, IsNil)

	err = s.waitOnActionSetState(c, as, crv1alpha1.StateRunning)
	c.Assert(err, IsNil)

	// Check if the func returned expected output
	c.Assert(testutil.OutputFuncOut(), DeepEquals, map[string]interface{}{"key": "myValue"})

	err = s.waitOnActionSetState(c, as, crv1alpha1.StateComplete)
	c.Assert(err, IsNil)

	// Check if the artifacts got updated correctly
	as, err = s.crCli.ActionSets(as.GetNamespace()).Get(as.GetName(), metav1.GetOptions{})
	arts := as.Status.Actions[0].Artifacts
	c.Assert(arts, NotNil)
	c.Assert(arts, HasLen, 1)
	keyVal := arts["myArt"].KeyValue
	c.Assert(keyVal, DeepEquals, map[string]string{"key": "myValue"})
}

func (s *ControllerSuite) TestRenderArtifactsFailure(c *C) {
	bp := newBPWithFakeOutputArtifact()
	bp = testutil.BlueprintWithConfigMap(bp)
	bp, err := s.crCli.Blueprints(s.namespace).Create(bp)
	c.Assert(err, IsNil)

	// Add an actionset that references that blueprint.
	as := testutil.NewTestActionSet(s.namespace, bp.GetName(), "Deployment", s.deployment.GetName(), s.namespace)
	as = testutil.ActionSetWithConfigMap(as, s.confimap.GetName())
	as, err = s.crCli.ActionSets(s.namespace).Create(as)
	c.Assert(err, IsNil)

	err = s.waitOnActionSetState(c, as, crv1alpha1.StateRunning)
	c.Assert(err, IsNil)

	c.Assert(testutil.OutputFuncOut(), DeepEquals, map[string]interface{}{"key": "myValue"})

	err = s.waitOnActionSetState(c, as, crv1alpha1.StateFailed)
	c.Assert(err, IsNil)
}
