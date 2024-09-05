// Copyright 2019 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	promgomodel "github.com/prometheus/client_model/go"
	. "gopkg.in/check.v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/client/clientset/versioned/scheme"
	crclientv1alpha1 "github.com/kanisterio/kanister/pkg/client/clientset/versioned/typed/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/eventer"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/function"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/poll"
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
	ss         *appsv1.StatefulSet
	deployment *appsv1.Deployment
	confimap   *corev1.ConfigMap
	recorder   record.EventRecorder
	ctrl       *Controller
}

var _ = Suite(&ControllerSuite{})

const (
	testAction = "myAction"
)

func (s *ControllerSuite) SetUpSuite(c *C) {
	config, err := kube.LoadConfig()
	c.Assert(err, IsNil)
	cli, err := kubernetes.NewForConfig(config)
	c.Assert(err, IsNil)
	crCli, err := crclientv1alpha1.NewForConfig(config)
	c.Assert(err, IsNil)

	// Make sure the CRD's exist.
	_ = resource.CreateCustomResources(context.Background(), config)

	s.cli = cli
	s.crCli = crCli

	s.recorder = eventer.NewEventRecorder(s.cli, "Controller Test")

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kanistercontrollertest-",
		},
	}
	ctx := context.Background()
	cns, err := s.cli.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	s.namespace = cns.Name

	sec := testutil.NewTestProfileSecret()
	sec, err = s.cli.CoreV1().Secrets(s.namespace).Create(ctx, sec, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	p := testutil.NewTestProfile(s.namespace, sec.GetName())
	_, err = s.crCli.Profiles(s.namespace).Create(ctx, p, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	ss := testutil.NewTestStatefulSet(1)
	ss, err = s.cli.AppsV1().StatefulSets(s.namespace).Create(ctx, ss, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	s.ss = ss
	err = kube.WaitOnStatefulSetReady(ctx, s.cli, s.namespace, s.ss.Name)
	c.Assert(err, IsNil)

	d := testutil.NewTestDeployment(1)
	d, err = s.cli.AppsV1().Deployments(s.namespace).Create(ctx, d, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	s.deployment = d
	err = kube.WaitOnDeploymentReady(ctx, s.cli, s.namespace, s.deployment.Name)
	c.Assert(err, IsNil)

	cm := testutil.NewTestConfigMap()
	cm, err = s.cli.CoreV1().ConfigMaps(s.namespace).Create(ctx, cm, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	s.confimap = cm
}

func (s *ControllerSuite) TearDownSuite(c *C) {
	err := os.Unsetenv(kube.PodNSEnvVar)
	c.Assert(err, IsNil)
	if s.namespace != "" {
		_ = s.cli.CoreV1().Namespaces().Delete(context.TODO(), s.namespace, metav1.DeleteOptions{})
	}
}

func (s *ControllerSuite) SetUpTest(c *C) {
	config, err := kube.LoadConfig()
	c.Assert(err, IsNil)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	testPrometheusRegistry := prometheus.NewRegistry()
	s.ctrl = New(config, testPrometheusRegistry)
	err = s.ctrl.StartWatch(ctx, s.namespace)
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
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	err := poll.Wait(ctx, func(context.Context) (bool, error) {
		as, err := s.crCli.ActionSets(as.GetNamespace()).Get(ctx, as.GetName(), metav1.GetOptions{})
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

func (s *ControllerSuite) waitOnDeferPhaseState(c *C, as *crv1alpha1.ActionSet, state crv1alpha1.State) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	err := poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		as, err := s.crCli.ActionSets(as.GetNamespace()).Get(ctx, as.GetName(), metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		if as.Status == nil {
			return false, fmt.Errorf("unexpected error: actionset status shouldn't be nil")
		}

		if as.Status.Actions[0].DeferPhase.State == state {
			return true, nil
		}

		return false, nil
	})
	if err == nil {
		return nil
	}
	return errors.Wrapf(err, "State '%s' never reached", state)
}

func (s *ControllerSuite) waitOnActionSetCompleteWithRunningPhases(as *crv1alpha1.ActionSet, rp *sets.Set[string]) error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	err := poll.Wait(ctx, func(context.Context) (bool, error) {
		as, err := s.crCli.ActionSets(as.GetNamespace()).Get(ctx, as.GetName(), metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if as.Status == nil {
			return false, nil
		}
		if as.Status.State == crv1alpha1.StateComplete {
			return true, nil
		}
		// These are non-terminal states.
		if as.Status.State == crv1alpha1.StatePending {
			return false, nil
		}
		if as.Status.State == crv1alpha1.StateRunning {
			// Delete running phases
			if rp.Has(as.Status.Progress.RunningPhase) {
				rp.Delete(as.Status.Progress.RunningPhase)
			}
			return false, nil
		}
		return false, errors.New(fmt.Sprintf("Unexpected state: %s", as.Status.State))
	})
	if err == nil {
		return nil
	}
	return errors.Wrapf(err, "ActionSet did not reach '%s' state", crv1alpha1.StateComplete)
}

func newBPWithOutputArtifact() *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-blueprint-",
		},
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"myAction": {
				OutputArtifacts: map[string]crv1alpha1.Artifact{
					"myArt": {
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

func newBPWithDeferPhaseAndErrInDeferPhase() *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-blueprint-defer-defererr-",
		},
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"backup": {
				OutputArtifacts: map[string]crv1alpha1.Artifact{
					"opArtPhaseOne": {
						KeyValue: map[string]string{
							"op": "{{ .Phases.backupPhaseOne.Output.value }}",
						},
					},
					"opArtPhaseTwo": {
						KeyValue: map[string]string{
							"op": "{{ .Phases.backupPhaseTwo.Output.value }}",
						},
					},
				},
				Phases: []crv1alpha1.BlueprintPhase{
					*phaseWithNameAndCMD("backupPhaseOne", []string{"kando", "output", "value", "mainValue"}),
					*phaseWithNameAndCMD("backupPhaseTwo", []string{"kando", "output", "value", "mainValueTwo"}),
				},
				DeferPhase: phaseWithNameAndCMD("deferPhase", []string{"exit", "1"}),
			},
		},
	}
}

func newBPWithDeferPhaseAndErrInCorePhase() *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-blueprint-defer-coreerr-",
		},
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"backup": {
				OutputArtifacts: map[string]crv1alpha1.Artifact{
					"opArtPhaesOne": {
						KeyValue: map[string]string{
							"op": "{{ .Phases.backupPhaseOne.Output.value }}",
						},
					},
					"opArtDeferPhase": {
						KeyValue: map[string]string{
							"op": "{{ .DeferPhase.Output.value }}",
						},
					},
				},
				Phases: []crv1alpha1.BlueprintPhase{
					*phaseWithNameAndCMD("backupPhaseOne", []string{"kando", "output", "value", "mainValue"}),
					*phaseWithNameAndCMD("backupPhaseTwo", []string{"exit", "1"}),
				},
				DeferPhase: phaseWithNameAndCMD("deferPhase", []string{"kando", "output", "value", "deferValue"}),
			},
		},
	}
}

// phaseWithNameAndCMD returns a phase that runs KubeTask function, with provided command
func phaseWithNameAndCMD(name string, command []string) *crv1alpha1.BlueprintPhase {
	return &crv1alpha1.BlueprintPhase{
		Name: name,
		Func: function.KubeTaskFuncName,
		Args: map[string]interface{}{
			"image":     consts.LatestKanisterToolsImage,
			"namespace": "default",
			"command":   command,
		},
	}
}

func newBPWithDeferPhase() *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-blueprint-defer-",
		},
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"backup": {
				// set output artifacts from main phases as well as deferPhase
				OutputArtifacts: map[string]crv1alpha1.Artifact{
					"mainPhaseOutputOne": {
						KeyValue: map[string]string{
							"op": "{{ .Phases.backupPhaseOne.Output.value }}",
						},
					},
					"mainPhaseOutputTwo": {
						KeyValue: map[string]string{
							"op": "{{ .Phases.backupPhaseTwo.Output.value }}",
						},
					},
					"deferPhaseOutput": {
						KeyValue: map[string]string{
							"op": "{{ .DeferPhase.Output.value }}",
						},
					},
				},
				Phases: []crv1alpha1.BlueprintPhase{
					*phaseWithNameAndCMD("backupPhaseOne", []string{"kando", "output", "value", "mainValue"}),
					*phaseWithNameAndCMD("backupPhaseTwo", []string{"kando", "output", "value", "mainValueTwo"}),
				},
				DeferPhase: phaseWithNameAndCMD("deferPhase", []string{"kando", "output", "value", "deferValue"}),
			},
			"restore": {
				InputArtifactNames: []string{
					"mainPhaseOutputOne",
					"mainPhaseOutputTwo",
					"deferPhaseOutput",
				},
				Phases: []crv1alpha1.BlueprintPhase{
					// this will try to render the output artifact that was set from all the phases of backup action
					*phaseWithNameAndCMD("restorePhase",
						[]string{"echo", "{{ .ArtifactsIn.deferPhaseOutput.KeyValue.op }}", "{{ .ArtifactsIn.mainPhaseOutputOne.KeyValue.op }}", "{{ .ArtifactsIn.mainPhaseOutputTwo.KeyValue.op }}"}),
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
			"myAction": {
				OutputArtifacts: map[string]crv1alpha1.Artifact{
					"myArt": {
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

func newBPWithKopiaSnapshotOutputArtifact() *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-blueprint-",
		},
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"myAction": {
				OutputArtifacts: map[string]crv1alpha1.Artifact{
					"myArt": {
						KopiaSnapshot: "{{ .Phases.myPhase0.Output.key }}",
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

func newBPForProgressRunningPhase() *crv1alpha1.Blueprint {
	return &crv1alpha1.Blueprint{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-bp-running-phase-",
		},
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"backup": {
				// set output artifacts from main phases as well as deferPhase
				OutputArtifacts: map[string]crv1alpha1.Artifact{},
				Phases: []crv1alpha1.BlueprintPhase{
					*phaseWithNameAndCMD("backupPhaseOne", []string{"sleep", "10"}),
					*phaseWithNameAndCMD("backupPhaseTwo", []string{"sleep", "8"}),
				},
				DeferPhase: phaseWithNameAndCMD("deferPhase", []string{"sleep", "8"}),
			},
		},
	}
}

func getCounterVecValue(metric prometheus.CounterVec, metricLabels []string) float64 {
	m := &promgomodel.Metric{}
	if err := metric.WithLabelValues(metricLabels...).Write(m); err != nil {
		return 0
	}
	return m.Counter.GetValue()
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
	as, err := s.crCli.ActionSets(s.namespace).Create(context.TODO(), as, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	defer func() {
		err := s.crCli.ActionSets(s.namespace).Delete(context.TODO(), as.GetName(), metav1.DeleteOptions{})
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
				{
					Object: crv1alpha1.ObjectReference{
						Name: "foo",
						Kind: param.NamespaceKind,
					},
					Blueprint: "NONEXISTANT_BLUEPRINT",
				},
			},
		},
	}
	as, err := s.crCli.ActionSets(s.namespace).Create(context.TODO(), as, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	defer func() {
		err := s.crCli.ActionSets(s.namespace).Delete(context.TODO(), as.GetName(), metav1.DeleteOptions{})
		c.Assert(err, IsNil)
	}()

	err = s.waitOnActionSetState(c, as, crv1alpha1.StateFailed)
	c.Assert(err, IsNil)
}

func (s *ControllerSuite) TestNilPrometheusRegistry(c *C) {
	config, err := kube.LoadConfig()
	c.Assert(err, IsNil)
	c.Assert(config, NotNil)
	ctrl := New(config, nil)
	c.Assert(ctrl, NotNil)
	c.Assert(ctrl.metrics, IsNil)
}

func (s *ControllerSuite) TestExecActionSet(c *C) {
	for _, pok := range []string{"StatefulSet", "Deployment"} {
		for _, tc := range []struct {
			funcNames        []string
			args             [][]string
			name             string
			version          string
			metricResolution string
		}{
			{
				funcNames:        []string{testutil.WaitFuncName},
				name:             "WaitFunc",
				version:          kanister.DefaultVersion,
				metricResolution: ActionSetCounterVecLabelResSuccess,
			},
			{
				funcNames:        []string{testutil.WaitFuncName, testutil.WaitFuncName},
				name:             "WaitWait",
				version:          kanister.DefaultVersion,
				metricResolution: ActionSetCounterVecLabelResSuccess,
			},
			{
				funcNames:        []string{testutil.FailFuncName},
				name:             "FailFunc",
				version:          kanister.DefaultVersion,
				metricResolution: ActionSetCounterVecLabelResFailure,
			},
			{
				funcNames:        []string{testutil.WaitFuncName, testutil.FailFuncName},
				name:             "WaitFail",
				version:          kanister.DefaultVersion,
				metricResolution: ActionSetCounterVecLabelResFailure,
			},
			{
				funcNames:        []string{testutil.FailFuncName, testutil.WaitFuncName},
				name:             "FailWait",
				version:          kanister.DefaultVersion,
				metricResolution: ActionSetCounterVecLabelResFailure,
			},
			{
				funcNames:        []string{testutil.ArgFuncName},
				name:             "ArgFunc",
				version:          kanister.DefaultVersion,
				metricResolution: ActionSetCounterVecLabelResSuccess,
			},
			{
				funcNames:        []string{testutil.ArgFuncName, testutil.FailFuncName},
				name:             "ArgFail",
				version:          kanister.DefaultVersion,
				metricResolution: ActionSetCounterVecLabelResFailure,
			},
			{
				funcNames:        []string{testutil.OutputFuncName},
				name:             "OutputFunc",
				version:          kanister.DefaultVersion,
				metricResolution: ActionSetCounterVecLabelResSuccess,
			},
			{
				funcNames:        []string{testutil.CancelFuncName},
				name:             "CancelFunc",
				version:          kanister.DefaultVersion,
				metricResolution: ActionSetCounterVecLabelResFailure,
			},
			{
				funcNames:        []string{testutil.ArgFuncName},
				name:             "ArgFuncVersion",
				version:          testutil.TestVersion,
				metricResolution: ActionSetCounterVecLabelResSuccess,
			},
			{
				funcNames:        []string{testutil.ArgFuncName},
				name:             "ArgFuncVersionFallback",
				version:          "v1.2.3",
				metricResolution: ActionSetCounterVecLabelResSuccess,
			},
			{
				funcNames:        []string{testutil.ArgFuncName},
				name:             "ArgFuncNoActionSetVersion",
				version:          "",
				metricResolution: ActionSetCounterVecLabelResSuccess,
			},
			{
				funcNames:        []string{testutil.VersionMismatchFuncName},
				name:             "VersionMismatchFunc",
				version:          "v1.2.3",
				metricResolution: ActionSetCounterVecLabelResFailure,
			},
			{
				funcNames:        []string{testutil.ArgFuncName, testutil.OutputFuncName},
				name:             "ArgOutputFallbackOnlyOutput",
				version:          testutil.TestVersion,
				metricResolution: ActionSetCounterVecLabelResSuccess,
			},
		} {
			var err error
			// Add a blueprint with a mocked kanister function.
			bp := testutil.NewTestBlueprint(pok, tc.funcNames...)
			bp = testutil.BlueprintWithConfigMap(bp)
			ctx := context.Background()
			bp, err = s.crCli.Blueprints(s.namespace).Create(ctx, bp, metav1.CreateOptions{})
			c.Assert(err, IsNil)

			oldValue := getCounterVecValue(s.ctrl.metrics.actionSetResolutionCounterVec, []string{tc.metricResolution})

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
			as := testutil.NewTestActionSet(s.namespace, bp.GetName(), pok, n, s.namespace, tc.version, testAction)
			as = testutil.ActionSetWithConfigMap(as, s.confimap.GetName())
			as, err = s.crCli.ActionSets(s.namespace).Create(ctx, as, metav1.CreateOptions{})
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
					testutil.CancelFuncStarted()
					err = s.crCli.ActionSets(s.namespace).Delete(context.TODO(), as.GetName(), metav1.DeleteOptions{})
					c.Assert(err, IsNil)
					c.Assert(testutil.CancelFuncOut().Error(), DeepEquals, "context canceled")
					cancel = true
				case testutil.VersionMismatchFuncName:
					final = crv1alpha1.StateFailed
					c.Assert(err, IsNil)
				}
			}

			if !cancel {
				err = s.waitOnActionSetState(c, as, final)
				c.Assert(err, IsNil, Commentf("Failed case: %s", tc.name))
				expectedValue := oldValue + 1
				err = waitForMetrics(s.ctrl.metrics.actionSetResolutionCounterVec, []string{tc.metricResolution}, expectedValue, time.Second)
				c.Assert(err, IsNil, Commentf("Failed case: %s, failed waiting for metric update to %v", tc.name, expectedValue))
			}
			err = s.crCli.Blueprints(s.namespace).Delete(context.TODO(), bp.GetName(), metav1.DeleteOptions{})
			c.Assert(err, IsNil)
			err = s.crCli.ActionSets(s.namespace).Delete(context.TODO(), as.GetName(), metav1.DeleteOptions{})
			if !cancel {
				c.Assert(err, IsNil)
			} else {
				c.Assert(err, NotNil)
			}
		}
	}
}

func waitForMetrics(metrics prometheus.CounterVec, labels []string, expected float64, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err := poll.Wait(ctx, func(context.Context) (bool, error) {
		current := getCounterVecValue(metrics, labels)
		if current == expected {
			return true, nil
		} else {
			return false, nil
		}
	})

	return err
}

func (s *ControllerSuite) TestRuntimeObjEventLogs(c *C) {
	c.Skip("This may not work in MiniKube")
	ctx := context.Background()
	// Create ActionSet
	as := &crv1alpha1.ActionSet{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "testactionset-",
		},
		Spec: &crv1alpha1.ActionSetSpec{
			Actions: []crv1alpha1.ActionSpec{
				{
					Blueprint: "NONEXISTANT_BLUEPRINT",
				},
			},
		},
	}
	as, err := s.crCli.ActionSets(s.namespace).Create(ctx, as, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	msg := "Unit testing event logs"
	reason := "Test Logs"

	// Create nil ActionSet
	var nilAs = (*crv1alpha1.ActionSet)(nil)

	// Create Blueprint
	bp := testutil.NewTestBlueprint("StatefulSet", testutil.WaitFuncName)
	bp, err = s.crCli.Blueprints(s.namespace).Create(ctx, bp, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	// Test the logAndErrorEvent function
	ctx = field.Context(ctx, consts.ActionsetNameKey, as.GetName())
	config, err := kube.LoadConfig()
	c.Assert(err, IsNil)
	ctlr := New(config, nil)
	ctlr.logAndErrorEvent(ctx, msg, reason, errors.New("Testing Event Logs"), as, nilAs, bp)

	// Test ActionSet error event logging
	events, err := s.cli.CoreV1().Events(as.Namespace).Search(scheme.Scheme, as)
	c.Assert(err, IsNil)
	c.Assert(events, NotNil)
	c.Assert(len(events.Items) > 0, Equals, true)
	c.Assert(events.Items[0].Message, Equals, msg)

	// Testing nil ActionSet error event logging
	events, err = s.cli.CoreV1().Events(as.Namespace).Search(scheme.Scheme, nilAs)
	c.Assert(err, NotNil)
	c.Assert(len(events.Items), Equals, 0)

	// Testing Blueprint error event logging
	events, err = s.cli.CoreV1().Events(bp.Namespace).Search(scheme.Scheme, bp)
	c.Assert(err, IsNil)
	c.Assert(events, NotNil)
	c.Assert(len(events.Items) > 0, Equals, true)
	c.Assert(events.Items[0].Message, Equals, msg)

	// Testing empty Blueprint
	testbp := &crv1alpha1.Blueprint{}
	ctlr.logAndErrorEvent(ctx, msg, reason, errors.New("Testing Event Logs"), testbp)
	events, err = s.cli.CoreV1().Events(bp.Namespace).Search(scheme.Scheme, testbp)
	c.Assert(err, NotNil)
	c.Assert(len(events.Items), Equals, 0)
}

func (s *ControllerSuite) TestDeferPhase(c *C) {
	err := os.Setenv(kube.PodNSEnvVar, "test")
	c.Assert(err, IsNil)

	ctx := context.Background()
	bp := newBPWithDeferPhase()

	bp, err = s.crCli.Blueprints(s.namespace).Create(ctx, bp, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	// create backup actionset and wait for it to be completed
	as := testutil.NewTestActionSet(s.namespace, bp.GetName(), "Deployment", s.deployment.GetName(), s.namespace, kanister.DefaultVersion, "backup")
	as, err = s.crCli.ActionSets(s.namespace).Create(ctx, as, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	err = s.waitOnActionSetState(c, as, crv1alpha1.StateRunning)
	c.Assert(err, IsNil)

	// make sure deferPhase is also run successfully
	err = s.waitOnDeferPhaseState(c, as, crv1alpha1.StateComplete)
	c.Assert(err, IsNil)

	err = s.waitOnActionSetState(c, as, crv1alpha1.StateComplete)
	c.Assert(err, IsNil)

	as, err = s.crCli.ActionSets(s.namespace).Get(ctx, as.Name, metav1.GetOptions{})
	c.Assert(err, IsNil)

	// create restore actionset and wait for it to be completed
	// if it's completed, simply means artifacts are rendered correctly
	ras := testutil.NewTestRestoreActionSet(s.namespace, bp.GetName(), s.deployment.GetName(), as.Status.Actions[0].Artifacts)
	ras, err = s.crCli.ActionSets(s.namespace).Create(ctx, ras, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	err = s.waitOnActionSetState(c, ras, crv1alpha1.StateRunning)
	c.Assert(err, IsNil)
	err = s.waitOnActionSetState(c, ras, crv1alpha1.StateComplete)
	c.Assert(err, IsNil)

	as, err = s.crCli.ActionSets(s.namespace).Get(ctx, as.Name, metav1.GetOptions{})
	c.Assert(err, IsNil)

	c.Assert(as.Status.Actions[0].Artifacts["mainPhaseOutputOne"].KeyValue, DeepEquals, map[string]string{"op": "mainValue"})
	c.Assert(as.Status.Actions[0].Artifacts["mainPhaseOutputTwo"].KeyValue, DeepEquals, map[string]string{"op": "mainValueTwo"})
	c.Assert(as.Status.Actions[0].Artifacts["deferPhaseOutput"].KeyValue, DeepEquals, map[string]string{"op": "deferValue"})
}

// TestDeferPhaseCoreErr tests a blueprint with multiple main phases and deferPhase
// since one of the main phases is returning error, we will have to make sure that
// 1. Actionset status is `failed`
// 2. DeferPhase is run successfully and status is complete
// 3. Phases have correct state in actionset status
// 4. We don't render output artifacts if any of the phases failed
func (s *ControllerSuite) TestDeferPhaseCoreErr(c *C) {
	err := os.Setenv(kube.PodNSEnvVar, "test")
	c.Assert(err, IsNil)
	ctx := context.Background()

	bp := newBPWithDeferPhaseAndErrInCorePhase()
	bp, err = s.crCli.Blueprints(s.namespace).Create(ctx, bp, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	as := testutil.NewTestActionSet(s.namespace, bp.GetName(), "Deployment", s.deployment.GetName(), s.namespace, kanister.DefaultVersion, "backup")
	as, err = s.crCli.ActionSets(s.namespace).Create(ctx, as, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	err = s.waitOnActionSetState(c, as, crv1alpha1.StateRunning)
	c.Assert(err, IsNil)

	// wait for deferPhase to be completed, because actionset status would be set to failed as soon as a main phase fails
	err = s.waitOnDeferPhaseState(c, as, crv1alpha1.StateComplete)
	c.Assert(err, IsNil)

	err = s.waitOnActionSetState(c, as, crv1alpha1.StateFailed)
	c.Assert(err, IsNil)

	// get the actionset again to have updated status
	as, err = s.crCli.ActionSets(s.namespace).Get(ctx, as.Name, metav1.GetOptions{})
	c.Assert(err, IsNil)

	// make sure the phases that errored have state to be se as failed in actionset status
	// since we just have backup action, we are using 0th index here
	c.Assert(as.Status.Actions[0].Phases[0].State, Equals, crv1alpha1.StateComplete)
	c.Assert(as.Status.Actions[0].Phases[1].State, Equals, crv1alpha1.StateFailed)
	c.Assert(as.Status.Actions[0].DeferPhase.State, Equals, crv1alpha1.StateComplete)

	// we don't render template if any of the core phases or defer phases failed
}

func (s *ControllerSuite) TestDeferPhaseDeferErr(c *C) {
	err := os.Setenv(kube.PodNSEnvVar, "test")
	c.Assert(err, IsNil)
	ctx := context.Background()

	bp := newBPWithDeferPhaseAndErrInDeferPhase()
	bp, err = s.crCli.Blueprints(s.namespace).Create(ctx, bp, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	as := testutil.NewTestActionSet(s.namespace, bp.GetName(), "Deployment", s.deployment.GetName(), s.namespace, kanister.DefaultVersion, "backup")
	as, err = s.crCli.ActionSets(s.namespace).Create(ctx, as, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	err = s.waitOnActionSetState(c, as, crv1alpha1.StateRunning)
	c.Assert(err, IsNil)

	// wait for deferPhase to fail
	err = s.waitOnDeferPhaseState(c, as, crv1alpha1.StateFailed)
	c.Assert(err, IsNil)

	err = s.waitOnActionSetState(c, as, crv1alpha1.StateFailed)
	c.Assert(err, IsNil)

	// get the actionset again to have updated status
	as, err = s.crCli.ActionSets(s.namespace).Get(ctx, as.Name, metav1.GetOptions{})
	c.Assert(err, IsNil)

	// make sure the phases that errored have state set as failed in actionset status
	// since we just have backup action, we are using 0th index here
	c.Assert(as.Status.Actions[0].Phases[0].State, Equals, crv1alpha1.StateComplete)
	c.Assert(as.Status.Actions[0].Phases[1].State, Equals, crv1alpha1.StateComplete)
	c.Assert(as.Status.Actions[0].DeferPhase.State, Equals, crv1alpha1.StateFailed)
}

func (s *ControllerSuite) TestPhaseOutputAsArtifact(c *C) {
	ctx := context.Background()
	// Create a blueprint that uses func output as artifact
	bp := newBPWithOutputArtifact()
	bp = testutil.BlueprintWithConfigMap(bp)
	bp, err := s.crCli.Blueprints(s.namespace).Create(ctx, bp, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	// Add an actionset that references that blueprint.
	as := testutil.NewTestActionSet(s.namespace, bp.GetName(), "Deployment", s.deployment.GetName(), s.namespace, kanister.DefaultVersion, testAction)
	as = testutil.ActionSetWithConfigMap(as, s.confimap.GetName())
	as, err = s.crCli.ActionSets(s.namespace).Create(ctx, as, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	err = s.waitOnActionSetState(c, as, crv1alpha1.StateRunning)
	c.Assert(err, IsNil)

	// Check if the func returned expected output
	c.Assert(testutil.OutputFuncOut(), DeepEquals, map[string]interface{}{"key": "myValue"})

	err = s.waitOnActionSetState(c, as, crv1alpha1.StateComplete)
	c.Assert(err, IsNil)

	// Check if the artifacts got updated correctly
	as, _ = s.crCli.ActionSets(as.GetNamespace()).Get(ctx, as.GetName(), metav1.GetOptions{})
	arts := as.Status.Actions[0].Artifacts
	c.Assert(arts, NotNil)
	c.Assert(arts, HasLen, 1)
	keyVal := arts["myArt"].KeyValue
	c.Assert(keyVal, DeepEquals, map[string]string{"key": "myValue"})
}

func (s *ControllerSuite) TestPhaseOutputParallelActions(c *C) {
	ctx := context.Background()
	// Create a blueprint that uses func output as artifact
	bp := newBPWithOutputArtifact()
	bp = testutil.BlueprintWithConfigMap(bp)
	bp, err := s.crCli.Blueprints(s.namespace).Create(ctx, bp, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	// Create another blueprint
	bp1 := testutil.NewTestBlueprint("Deployment", testutil.WaitFuncName)
	bp1, err = s.crCli.Blueprints(s.namespace).Create(ctx, bp1, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	// Add an actionset that runs actions from two blueprints in parallel
	as := testutil.NewTestMultiActionActionSet(s.namespace, bp1.GetName(), testAction, bp.GetName(), testAction, "Deployment", s.deployment.GetName(), s.namespace, kanister.DefaultVersion)
	as = testutil.ActionSetWithConfigMap(as, s.confimap.GetName())
	as, err = s.crCli.ActionSets(s.namespace).Create(ctx, as, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	err = s.waitOnActionSetState(c, as, crv1alpha1.StateRunning)
	c.Assert(err, IsNil)

	// Check if the func returned expected output
	c.Assert(testutil.OutputFuncOut(), DeepEquals, map[string]interface{}{"key": "myValue"})

	testutil.ReleaseWaitFunc()

	err = s.waitOnActionSetState(c, as, crv1alpha1.StateComplete)
	c.Assert(err, IsNil)

	// Check if the artifacts got updated correctly
	as, _ = s.crCli.ActionSets(as.GetNamespace()).Get(ctx, as.GetName(), metav1.GetOptions{})
	arts := as.Status.Actions[0].Artifacts
	c.Assert(arts, IsNil)

	arts = as.Status.Actions[1].Artifacts
	c.Assert(arts, NotNil)
	c.Assert(arts, HasLen, 1)
	keyVal := arts["myArt"].KeyValue
	c.Assert(keyVal, DeepEquals, map[string]string{"key": "myValue"})
}

func (s *ControllerSuite) TestPhaseOutputAsKopiaSnapshot(c *C) {
	ctx := context.Background()
	// Create a blueprint that uses func output as kopia snapshot
	bp := newBPWithKopiaSnapshotOutputArtifact()
	bp = testutil.BlueprintWithConfigMap(bp)
	bp, err := s.crCli.Blueprints(s.namespace).Create(ctx, bp, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	// Add an actionset that references that blueprint.
	as := testutil.NewTestActionSet(s.namespace, bp.GetName(), "Deployment", s.deployment.GetName(), s.namespace, kanister.DefaultVersion, testAction)
	as = testutil.ActionSetWithConfigMap(as, s.confimap.GetName())
	as, err = s.crCli.ActionSets(s.namespace).Create(ctx, as, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	err = s.waitOnActionSetState(c, as, crv1alpha1.StateRunning)
	c.Assert(err, IsNil)

	// Check if the func returned expected output
	c.Assert(testutil.OutputFuncOut(), DeepEquals, map[string]interface{}{"key": "myValue"})

	err = s.waitOnActionSetState(c, as, crv1alpha1.StateComplete)
	c.Assert(err, IsNil)

	// Check if the artifacts got updated correctly
	as, _ = s.crCli.ActionSets(as.GetNamespace()).Get(ctx, as.GetName(), metav1.GetOptions{})
	arts := as.Status.Actions[0].Artifacts
	c.Assert(arts, NotNil)
	c.Assert(arts, HasLen, 1)
	kopiaSnapshot := arts["myArt"].KopiaSnapshot
	c.Assert(kopiaSnapshot, Equals, "myValue")
}

func (s *ControllerSuite) TestActionSetExecWithoutProfile(c *C) {
	ctx := context.Background()
	// Create a blueprint that uses func output as artifact
	bp := newBPWithOutputArtifact()
	bp = testutil.BlueprintWithConfigMap(bp)
	bp, err := s.crCli.Blueprints(s.namespace).Create(ctx, bp, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	// Add an actionset that references that blueprint.
	as := &crv1alpha1.ActionSet{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-actionset-",
			Namespace:    s.namespace,
		},
		Spec: &crv1alpha1.ActionSetSpec{
			Actions: []crv1alpha1.ActionSpec{
				{
					Blueprint: bp.GetName(),
					Name:      "myAction",
					Object: crv1alpha1.ObjectReference{
						Kind:      "Deployment",
						Name:      s.deployment.GetName(),
						Namespace: s.namespace,
					},
					PreferredVersion: kanister.DefaultVersion,
				},
			},
		},
	}
	as = testutil.ActionSetWithConfigMap(as, s.confimap.GetName())
	as, err = s.crCli.ActionSets(s.namespace).Create(ctx, as, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	err = s.waitOnActionSetState(c, as, crv1alpha1.StateRunning)
	c.Assert(err, IsNil)

	// Check if the func returned expected output
	c.Assert(testutil.OutputFuncOut(), DeepEquals, map[string]interface{}{"key": "myValue"})

	err = s.waitOnActionSetState(c, as, crv1alpha1.StateComplete)
	c.Assert(err, IsNil)

	// Check if the artifacts got updated correctly
	as, _ = s.crCli.ActionSets(as.GetNamespace()).Get(ctx, as.GetName(), metav1.GetOptions{})
	arts := as.Status.Actions[0].Artifacts
	c.Assert(arts, NotNil)
	c.Assert(arts, HasLen, 1)
	keyVal := arts["myArt"].KeyValue
	c.Assert(keyVal, DeepEquals, map[string]string{"key": "myValue"})
}

func (s *ControllerSuite) TestRenderArtifactsFailure(c *C) {
	ctx := context.Background()
	bp := newBPWithFakeOutputArtifact()
	bp = testutil.BlueprintWithConfigMap(bp)
	bp, err := s.crCli.Blueprints(s.namespace).Create(ctx, bp, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	// Add an actionset that references that blueprint.
	as := testutil.NewTestActionSet(s.namespace, bp.GetName(), "Deployment", s.deployment.GetName(), s.namespace, kanister.DefaultVersion, testAction)
	as = testutil.ActionSetWithConfigMap(as, s.confimap.GetName())
	as, err = s.crCli.ActionSets(s.namespace).Create(ctx, as, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	err = s.waitOnActionSetState(c, as, crv1alpha1.StateRunning)
	c.Assert(err, IsNil)

	c.Assert(testutil.OutputFuncOut(), DeepEquals, map[string]interface{}{"key": "myValue"})

	err = s.waitOnActionSetState(c, as, crv1alpha1.StateFailed)
	c.Assert(err, IsNil)
}

func (s *ControllerSuite) TestProgressRunningPhase(c *C) {
	err := os.Setenv(kube.PodNSEnvVar, "test")
	c.Assert(err, IsNil)
	ctx := context.Background()

	bp := newBPForProgressRunningPhase()
	bp, err = s.crCli.Blueprints(s.namespace).Create(ctx, bp, metav1.CreateOptions{})
	c.Assert(err, IsNil)

	// create actionset and wait for it to reach Running state
	as := testutil.NewTestActionSet(s.namespace, bp.GetName(), "Deployment", s.deployment.GetName(), s.namespace, kanister.DefaultVersion, "backup")
	as, err = s.crCli.ActionSets(s.namespace).Create(ctx, as, metav1.CreateOptions{})
	c.Assert(err, IsNil)
	err = s.waitOnActionSetState(c, as, crv1alpha1.StateRunning)
	c.Assert(err, IsNil)

	runningPhases := sets.Set[string]{}
	runningPhases.Insert("backupPhaseOne").Insert("backupPhaseTwo").Insert("deferPhase")

	err = s.waitOnActionSetCompleteWithRunningPhases(as, &runningPhases)
	c.Assert(err, IsNil)
	c.Assert(runningPhases, HasLen, 0)
}

func (s *ControllerSuite) TestGetActionTypeBucket(c *C) {
	for _, tc := range []struct {
		actionType string
	}{
		{
			actionType: ActionTypeBackup,
		},
		{
			actionType: ActionTypeRestore,
		},
		{
			actionType: ActionTypeDelete,
		},
		{
			actionType: ActionTypeBackupToServer,
		},
		{
			actionType: ActionTypeRestoreFromServer,
		},
		{
			actionType: ActionTypeBeforeBackup,
		},
		{
			actionType: ActionTypeOnSuccess,
		},
		{
			actionType: ActionTypeOnFailure,
		},
		{
			actionType: ActionTypePreRestore,
		},
		{
			actionType: ActionTypePostRestore,
		},
		{
			actionType: ActionTypePostRestoreFailed,
		},
		{
			actionType: ActionTypeBackupPrehook,
		},
		{
			actionType: ActionTypeBackupPosthook,
		},
		{
			actionType: "random-action",
		},
	} {
		switch tc.actionType {
		case ActionTypeBackup:
			c.Assert(getActionTypeBucket(tc.actionType), Equals, ActionTypeBackup)
		case ActionTypeRestore:
			c.Assert(getActionTypeBucket(tc.actionType), Equals, ActionTypeRestore)
		case ActionTypeDelete:
			c.Assert(getActionTypeBucket(tc.actionType), Equals, ActionTypeDelete)
		case ActionTypeBackupToServer:
			c.Assert(getActionTypeBucket(tc.actionType), Equals, ActionTypeBackupToServer)
		case ActionTypeRestoreFromServer:
			c.Assert(getActionTypeBucket(tc.actionType), Equals, ActionTypeRestoreFromServer)
		case ActionTypeBeforeBackup:
			c.Assert(getActionTypeBucket(tc.actionType), Equals, ActionTypeBeforeBackup)
		case ActionTypeOnSuccess:
			c.Assert(getActionTypeBucket(tc.actionType), Equals, ActionTypeOnSuccess)
		case ActionTypeOnFailure:
			c.Assert(getActionTypeBucket(tc.actionType), Equals, ActionTypeOnFailure)
		case ActionTypePreRestore:
			c.Assert(getActionTypeBucket(tc.actionType), Equals, ActionTypePreRestore)
		case ActionTypePostRestore:
			c.Assert(getActionTypeBucket(tc.actionType), Equals, ActionTypePostRestore)
		case ActionTypePostRestoreFailed:
			c.Assert(getActionTypeBucket(tc.actionType), Equals, ActionTypePostRestoreFailed)
		case ActionTypeBackupPrehook:
			c.Assert(getActionTypeBucket(tc.actionType), Equals, ActionTypeBackupPrehook)
		case ActionTypeBackupPosthook:
			c.Assert(getActionTypeBucket(tc.actionType), Equals, ActionTypeBackupPosthook)
		default:
			c.Assert(getActionTypeBucket(tc.actionType), Equals, ActionTypeBackupOther)
		}
	}
}
