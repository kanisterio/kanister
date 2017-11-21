package controller

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pkg/errors"
	. "gopkg.in/check.v1"
	"k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type ControllerSuite struct {
	crCli      crv1alpha1.CRV1alpha1Interface
	cli        kubernetes.Interface
	namespace  string
	cancel     func()
	waitFuncCh chan struct{}
	argFuncCh  chan []string
}

var _ = Suite(&ControllerSuite{})

const (
	waitFuncName = "WaitFunc"
	failFuncName = "FailFunc"
	argFuncName  = "ArgFunc"
)

const (
	actionSetName       = "testactionset"
	blueprintName       = "testblueprint"
	protectedObjectName = "testprotectedobject"
	configMapName       = "testconfigmap"
)

func (s *ControllerSuite) SetUpSuite(c *C) {
	config, err := kube.LoadConfig()
	c.Assert(err, IsNil)
	cli, err := kubernetes.NewForConfig(config)
	c.Assert(err, IsNil)
	crCli, err := crv1alpha1.NewForConfig(config)
	c.Assert(err, IsNil)

	s.cli = cli
	s.crCli = crCli

	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kanistercontrollertest-",
		},
	}
	cns, err := s.cli.Core().Namespaces().Create(ns)
	c.Assert(err, IsNil)
	s.namespace = cns.Name

	s.waitFuncCh = make(chan struct{})
	registerMockKanisterFunc(waitFuncName, func(context.Context, ...string) error {
		c.Logf("Running Kanister Function: %s", waitFuncName)
		<-s.waitFuncCh
		return nil
	})
	registerMockKanisterFunc(failFuncName, func(context.Context, ...string) error {
		c.Logf("Running Kanister function: %s", failFuncName)
		return errors.New("Kanister Function Failed")
	})
	s.argFuncCh = make(chan []string)
	registerMockKanisterFunc(argFuncName, func(ctx context.Context, args ...string) error {
		c.Logf("Running Kanister function: %s: %#v", argFuncName, args)
		s.argFuncCh <- args
		return nil
	})

	ss := newTestStatefulSet()
	_, err = s.cli.AppsV1beta1().StatefulSets(s.namespace).Create(ss)
	c.Assert(err, IsNil)

	d := newTestDeployment()
	_, err = s.cli.AppsV1beta1().Deployments(s.namespace).Create(d)
	c.Assert(err, IsNil)

	cm := newTestConfigMap()
	_, err = s.cli.CoreV1().ConfigMaps(s.namespace).Create(cm)
	c.Assert(err, IsNil)

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

var _ kanister.Func = (*mockKanisterFunc)(nil)

type mockKanisterFunc struct {
	name string
	f    func(context.Context, ...string) error
}

func (mf *mockKanisterFunc) Exec(ctx context.Context, args ...string) error {
	return mf.f(ctx, args...)
}

func (mf *mockKanisterFunc) Name() string {
	return mf.name
}
func registerMockKanisterFunc(name string, f func(context.Context, ...string) error) {
	kanister.Register(&mockKanisterFunc{name: name, f: f})
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
			Name: actionSetName,
		},
		Spec: &crv1alpha1.ActionSetSpec{
			Actions: []crv1alpha1.ActionSpec{},
		},
	}
	as, err := s.crCli.ActionSets(s.namespace).Create(as)
	c.Assert(err, IsNil)
	defer func() {
		err := s.crCli.ActionSets(s.namespace).Delete(actionSetName, nil)
		c.Assert(err, IsNil)
	}()

	err = s.waitOnActionSetState(c, as, crv1alpha1.StateComplete)
	c.Assert(err, IsNil)
}

func newTestDeployment() *v1beta1.Deployment {
	var replicas int32 = 1
	return &v1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: protectedObjectName,
		},
		Spec: v1beta1.DeploymentSpec{
			Replicas: &replicas,
			Template: newTestPodTemplateSpec(),
		},
	}
}

func newTestStatefulSet() *v1beta1.StatefulSet {
	var replicas int32 = 1
	return &v1beta1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: protectedObjectName,
		},
		Spec: v1beta1.StatefulSetSpec{
			Replicas:    &replicas,
			ServiceName: "fake-svc",
			Template:    newTestPodTemplateSpec(),
		},
	}
}

func newTestPodTemplateSpec() v1.PodTemplateSpec {
	return v1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"app": "fake-app",
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				v1.Container{
					Name:    "test-container",
					Image:   "alpine:3.6",
					Command: []string{"tail"},
					Args:    []string{"-f", "/dev/null"},
				},
			},
		},
	}
}

func newTestActionSet(namespace string, poKind string) *crv1alpha1.ActionSet {
	return &crv1alpha1.ActionSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      actionSetName,
			Namespace: namespace,
		},
		Spec: &crv1alpha1.ActionSetSpec{
			Actions: []crv1alpha1.ActionSpec{
				crv1alpha1.ActionSpec{
					Blueprint: blueprintName,
					Name:      "myAction",
					Object: crv1alpha1.ObjectReference{
						Kind:      poKind,
						Name:      protectedObjectName,
						Namespace: namespace,
					},
				},
			},
		},
	}
}

func newTestConfigMap() *v1.ConfigMap {
	return &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: configMapName,
		},
		Data: map[string]string{
			"myKey": "myValue",
		},
	}
}

const actionName = "myAction"

func newTestBlueprint(phaseFuncs ...string) *crv1alpha1.Blueprint {
	bp := &crv1alpha1.Blueprint{
		ObjectMeta: metav1.ObjectMeta{
			Name: blueprintName,
		},
		Actions: map[string]*crv1alpha1.BlueprintAction{
			actionName: &crv1alpha1.BlueprintAction{
				Kind:   "statefulset",
				Phases: make([]crv1alpha1.BlueprintPhase, 0, len(phaseFuncs)),
			},
		},
	}
	for _, bpa := range bp.Actions {
		for i, pf := range phaseFuncs {
			bpa.Phases = append(bpa.Phases, crv1alpha1.BlueprintPhase{
				Name: fmt.Sprintf("myPhase-%d", i),
				Func: pf,
			})
		}
	}
	return bp
}

func actionSetWithConfigMap(as *crv1alpha1.ActionSet) *crv1alpha1.ActionSet {
	as.Spec.Actions[0].ConfigMaps = map[string]crv1alpha1.ObjectReference{
		"myCM": crv1alpha1.ObjectReference{
			Name:      configMapName,
			Namespace: as.GetNamespace(),
		},
	}
	return as
}

func blueprintWithConfigMap(bp *crv1alpha1.Blueprint) *crv1alpha1.Blueprint {
	cmArgs := []string{"{{ .ConfigMaps.myCM.Data.myKey  }}"}
	bp.Actions[actionName].Phases[0].Args = cmArgs
	return bp
}

func (s *ControllerSuite) TestExecActionSet(c *C) {

	for _, pok := range []string{"statefulset", "deployment"} {
		for _, tc := range []struct {
			bp      *crv1alpha1.Blueprint
			as      *crv1alpha1.ActionSet
			waits   int
			args    [][]string
			outcome crv1alpha1.State
		}{
			{
				bp:      newTestBlueprint(waitFuncName),
				as:      newTestActionSet(s.namespace, pok),
				waits:   1,
				outcome: crv1alpha1.StateComplete,
			},
			{
				bp:      newTestBlueprint(waitFuncName, waitFuncName),
				as:      newTestActionSet(s.namespace, pok),
				waits:   2,
				outcome: crv1alpha1.StateComplete,
			},
			{
				bp:      newTestBlueprint(failFuncName),
				as:      newTestActionSet(s.namespace, pok),
				outcome: crv1alpha1.StateFailed,
			},
			{
				bp:      blueprintWithConfigMap(newTestBlueprint(argFuncName)),
				as:      actionSetWithConfigMap(newTestActionSet(s.namespace, pok)),
				args:    [][]string{{"myValue"}},
				outcome: crv1alpha1.StateComplete,
			},
		} {
			// Add a blueprint with a mocked kanister function.
			_, err := s.crCli.Blueprints(s.namespace).Create(tc.bp)
			c.Assert(err, IsNil)

			// Add an actionset that references that blueprint.
			as, err := s.crCli.ActionSets(s.namespace).Create(tc.as)
			c.Assert(err, IsNil)

			err = s.waitOnActionSetState(c, as, crv1alpha1.StateRunning)
			c.Assert(err, IsNil)

			for i := 0; i < tc.waits; i++ {
				s.waitFuncCh <- struct{}{}
			}

			for _, args := range tc.args {
				c.Assert(<-s.argFuncCh, DeepEquals, args)
			}

			err = s.waitOnActionSetState(c, as, tc.outcome)
			c.Assert(err, IsNil)

			// Clean up temporary objects.
			err = s.crCli.Blueprints(s.namespace).Delete(blueprintName, nil)
			c.Assert(err, IsNil)
			err = s.crCli.ActionSets(s.namespace).Delete(actionSetName, nil)
			c.Assert(err, IsNil)
		}
	}
}
