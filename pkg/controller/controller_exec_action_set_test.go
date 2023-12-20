// This is a split from controller_test.go in order to allow
// parallel execution of test suites
package controller

import (
	"context"
	"time"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/poll"
	"github.com/kanisterio/kanister/pkg/testutil"
	"github.com/prometheus/client_golang/prometheus"
	. "gopkg.in/check.v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ControllerExecActionSetSuite struct {
	base ControllerSuite
}

var _ = Suite(&ControllerExecActionSetSuite{})

// This suite is a split from controller_test.go
// We use setup and teardown functions from ControllerSuite
func (s *ControllerExecActionSetSuite) SetUpSuite(c *C) {
	s.base.SetUpSuite(c)
}
func (s *ControllerExecActionSetSuite) TearDownSuite(c *C) {
	s.base.TearDownSuite(c)
}
func (s *ControllerExecActionSetSuite) SetUpTest(c *C) {
	s.base.SetUpTest(c)
}
func (s *ControllerExecActionSetSuite) TearDownTest(c *C) {
	s.base.TearDownTest(c)
}

func (s *ControllerExecActionSetSuite) TestExecActionSet(c *C) {
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
				metricResolution: ACTION_SET_COUNTER_VEC_LABEL_RES_SUCCESS,
			},
			{
				funcNames:        []string{testutil.WaitFuncName, testutil.WaitFuncName},
				name:             "WaitWait",
				version:          kanister.DefaultVersion,
				metricResolution: ACTION_SET_COUNTER_VEC_LABEL_RES_SUCCESS,
			},
			{
				funcNames:        []string{testutil.FailFuncName},
				name:             "FailFunc",
				version:          kanister.DefaultVersion,
				metricResolution: ACTION_SET_COUNTER_VEC_LABEL_RES_FAILURE,
			},
			{
				funcNames:        []string{testutil.WaitFuncName, testutil.FailFuncName},
				name:             "WaitFail",
				version:          kanister.DefaultVersion,
				metricResolution: ACTION_SET_COUNTER_VEC_LABEL_RES_FAILURE,
			},
			{
				funcNames:        []string{testutil.FailFuncName, testutil.WaitFuncName},
				name:             "FailWait",
				version:          kanister.DefaultVersion,
				metricResolution: ACTION_SET_COUNTER_VEC_LABEL_RES_FAILURE,
			},
			{
				funcNames:        []string{testutil.ArgFuncName},
				name:             "ArgFunc",
				version:          kanister.DefaultVersion,
				metricResolution: ACTION_SET_COUNTER_VEC_LABEL_RES_SUCCESS,
			},
			{
				funcNames:        []string{testutil.ArgFuncName, testutil.FailFuncName},
				name:             "ArgFail",
				version:          kanister.DefaultVersion,
				metricResolution: ACTION_SET_COUNTER_VEC_LABEL_RES_FAILURE,
			},
			{
				funcNames:        []string{testutil.OutputFuncName},
				name:             "OutputFunc",
				version:          kanister.DefaultVersion,
				metricResolution: ACTION_SET_COUNTER_VEC_LABEL_RES_SUCCESS,
			},
			{
				funcNames:        []string{testutil.CancelFuncName},
				name:             "CancelFunc",
				version:          kanister.DefaultVersion,
				metricResolution: ACTION_SET_COUNTER_VEC_LABEL_RES_FAILURE,
			},
			{
				funcNames:        []string{testutil.ArgFuncName},
				name:             "ArgFuncVersion",
				version:          testutil.TestVersion,
				metricResolution: ACTION_SET_COUNTER_VEC_LABEL_RES_SUCCESS,
			},
			{
				funcNames:        []string{testutil.ArgFuncName},
				name:             "ArgFuncVersionFallback",
				version:          "v1.2.3",
				metricResolution: ACTION_SET_COUNTER_VEC_LABEL_RES_SUCCESS,
			},
			{
				funcNames:        []string{testutil.ArgFuncName},
				name:             "ArgFuncNoActionSetVersion",
				version:          "",
				metricResolution: ACTION_SET_COUNTER_VEC_LABEL_RES_SUCCESS,
			},
			{
				funcNames:        []string{testutil.VersionMismatchFuncName},
				name:             "VersionMismatchFunc",
				version:          "v1.2.3",
				metricResolution: ACTION_SET_COUNTER_VEC_LABEL_RES_FAILURE,
			},
			{
				funcNames:        []string{testutil.ArgFuncName, testutil.OutputFuncName},
				name:             "ArgOutputFallbackOnlyOutput",
				version:          testutil.TestVersion,
				metricResolution: ACTION_SET_COUNTER_VEC_LABEL_RES_SUCCESS,
			},
		} {
			var err error
			// Add a blueprint with a mocked kanister function.
			bp := testutil.NewTestBlueprint(pok, tc.funcNames...)
			bp = testutil.BlueprintWithConfigMap(bp)
			ctx := context.Background()
			bp, err = s.base.crCli.Blueprints(s.base.namespace).Create(ctx, bp, metav1.CreateOptions{})
			c.Assert(err, IsNil)

			oldValue := getCounterVecValue(s.base.ctrl.metrics.actionSetResolutionCounterVec, []string{tc.metricResolution})

			var n string
			switch pok {
			case "StatefulSet":
				n = s.base.ss.GetName()
			case "Deployment":
				n = s.base.deployment.GetName()
			default:
				c.FailNow()
			}

			// Add an actionset that references that blueprint.
			as := testutil.NewTestActionSet(s.base.namespace, bp.GetName(), pok, n, s.base.namespace, tc.version, testAction)
			as = testutil.ActionSetWithConfigMap(as, s.base.confimap.GetName())
			as, err = s.base.crCli.ActionSets(s.base.namespace).Create(ctx, as, metav1.CreateOptions{})
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
					err = s.base.crCli.ActionSets(s.base.namespace).Delete(context.TODO(), as.GetName(), metav1.DeleteOptions{})
					c.Assert(err, IsNil)
					c.Assert(testutil.CancelFuncOut().Error(), DeepEquals, "context canceled")
					cancel = true
				case testutil.VersionMismatchFuncName:
					final = crv1alpha1.StateFailed
					c.Assert(err, IsNil)
				}
			}

			if !cancel {
				err = s.base.waitOnActionSetState(c, as, final)
				c.Assert(err, IsNil, Commentf("Failed case: %s", tc.name))
				expectedValue := oldValue + 1
				err = waitForMetrics(s.base.ctrl.metrics.actionSetResolutionCounterVec, []string{tc.metricResolution}, expectedValue, time.Second)
				c.Assert(err, IsNil, Commentf("Failed case: %s, failed waiting for metric update to %v", tc.name, expectedValue))
			}
			err = s.base.crCli.Blueprints(s.base.namespace).Delete(context.TODO(), bp.GetName(), metav1.DeleteOptions{})
			c.Assert(err, IsNil)
			err = s.base.crCli.ActionSets(s.base.namespace).Delete(context.TODO(), as.GetName(), metav1.DeleteOptions{})
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
