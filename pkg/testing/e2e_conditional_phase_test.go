//go:build integration
// +build integration

// Copyright 2026 The Kanister Authors.
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

package testing

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"gopkg.in/check.v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/function"
	"github.com/kanisterio/kanister/pkg/kanctl"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/poll"
	"github.com/kanisterio/kanister/pkg/testutil"
)

const conditionalPhaseTestTimeout = 2 * time.Minute

// conditionalPhaseTarget holds the objects every conditional phase test needs.
type conditionalPhaseTarget struct {
	deployment string
	profile    string
}

// setUpConditionalPhaseTarget creates a ready Deployment, and a Secret and
// Profile for ActionSets to reference.
func (s *E2ESuite) setUpConditionalPhaseTarget(ctx context.Context, c *check.C) conditionalPhaseTarget {
	d, err := s.cli.AppsV1().Deployments(s.namespace).Create(ctx, testutil.NewTestDeployment(1), metav1.CreateOptions{})
	c.Assert(err, check.IsNil)
	err = kube.WaitOnDeploymentReady(ctx, s.cli, s.namespace, d.GetName())
	c.Assert(err, check.IsNil)

	sec := testutil.NewTestProfileSecret()
	sec, err = s.cli.CoreV1().Secrets(s.namespace).Create(ctx, sec, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)

	// NewTestProfile uses a fixed name, which would clash with the Profile
	// created by other tests in this suite's shared namespace.
	p := testutil.NewTestProfile(s.namespace, sec.GetName())
	p.Name = ""
	p.GenerateName = "test-profile-"
	p, err = s.crCli.Profiles(s.namespace).Create(ctx, p, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)

	return conditionalPhaseTarget{deployment: d.GetName(), profile: p.GetName()}
}

// runConditionalActionSet creates a Blueprint with the given phases, runs its
// only action and waits for the ActionSet to reach a terminal state, either
// complete or failed. It returns the final ActionSet.
func (s *E2ESuite) runConditionalActionSet(
	ctx context.Context,
	c *check.C,
	t conditionalPhaseTarget,
	phases []crv1alpha1.BlueprintPhase,
	options map[string]string,
) *crv1alpha1.ActionSet {
	bpName := s.createConditionalBlueprint(ctx, c, map[string]*crv1alpha1.BlueprintAction{
		"test": {
			Kind:   "Deployment",
			Phases: phases,
		},
	})
	return s.runConditionalAction(ctx, c, t, bpName, "test", options)
}

// createConditionalBlueprint creates a Blueprint with the given actions and
// returns its name.
func (s *E2ESuite) createConditionalBlueprint(
	ctx context.Context,
	c *check.C,
	actions map[string]*crv1alpha1.BlueprintAction,
) string {
	bp := &crv1alpha1.Blueprint{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-blueprint-",
		},
		Actions: actions,
	}
	bp, err := s.crCli.Blueprints(s.namespace).Create(ctx, bp, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)
	return bp.GetName()
}

// runConditionalAction runs the named action of an existing Blueprint against
// the test Deployment and waits for it to reach a terminal state.
func (s *E2ESuite) runConditionalAction(
	ctx context.Context,
	c *check.C,
	t conditionalPhaseTarget,
	blueprint, action string,
	options map[string]string,
) *crv1alpha1.ActionSet {
	as := &crv1alpha1.ActionSet{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-actionset-",
		},
		Spec: &crv1alpha1.ActionSetSpec{
			Actions: []crv1alpha1.ActionSpec{
				{
					Name: action,
					Object: crv1alpha1.ObjectReference{
						Kind:      "Deployment",
						Name:      t.deployment,
						Namespace: s.namespace,
					},
					Blueprint: blueprint,
					Profile: &crv1alpha1.ObjectReference{
						Name:      t.profile,
						Namespace: s.namespace,
					},
					Options: options,
				},
			},
		},
	}
	return s.createAndWaitForActionSet(ctx, c, as)
}

// runChildActionSet derives a child of parent for the named action, applies
// the option overrides, and waits for it to reach a terminal state.
func (s *E2ESuite) runChildActionSet(
	ctx context.Context,
	c *check.C,
	parent *crv1alpha1.ActionSet,
	action string,
	options map[string]string,
) *crv1alpha1.ActionSet {
	child, err := kanctl.ChildActionSet(parent, &kanctl.PerformParams{
		ActionName: action,
		ParentName: parent.GetName(),
		Options:    options,
	})
	c.Assert(err, check.IsNil)
	return s.createAndWaitForActionSet(ctx, c, child)
}

// createAndWaitForActionSet creates the ActionSet and waits for it to reach a
// terminal state, either complete or failed. It returns the final ActionSet.
func (s *E2ESuite) createAndWaitForActionSet(
	ctx context.Context,
	c *check.C,
	as *crv1alpha1.ActionSet,
) *crv1alpha1.ActionSet {
	as, err := s.crCli.ActionSets(s.namespace).Create(ctx, as, metav1.CreateOptions{})
	c.Assert(err, check.IsNil)

	err = poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		as, err = s.crCli.ActionSets(s.namespace).Get(ctx, as.GetName(), metav1.GetOptions{})
		switch {
		case err != nil, as.Status == nil:
			return false, err
		case as.Status.State == crv1alpha1.StateFailed, as.Status.State == crv1alpha1.StateComplete:
			return true, nil
		}
		return false, nil
	})
	c.Assert(err, check.IsNil)
	return as
}

// execPhase returns a KubeExec phase that runs cmd in the test Deployment's pod.
func execPhase(name, condition string, cmd ...string) crv1alpha1.BlueprintPhase {
	return crv1alpha1.BlueprintPhase{
		Func: function.KubeExecFuncName,
		Name: name,
		If:   condition,
		Args: map[string]interface{}{
			"namespace": "{{ .Deployment.Namespace }}",
			"pod":       "{{ index .Deployment.Pods 0 }}",
			"container": "test-container",
			"command":   cmd,
		},
	}
}

// outputPhase returns a KubeTask phase that publishes key=value as phase output.
func outputPhase(name, condition, key, value string) crv1alpha1.BlueprintPhase {
	return crv1alpha1.BlueprintPhase{
		Func: function.KubeTaskFuncName,
		Name: name,
		If:   condition,
		Args: map[string]interface{}{
			"image":     consts.LatestKanisterToolsImage,
			"namespace": "{{ .Deployment.Namespace }}",
			"command":   []string{"kando", "output", key, value},
			"podOverride": map[string]interface{}{
				"containers": []map[string]interface{}{
					{
						"name":            "container",
						"imagePullPolicy": "IfNotPresent",
					},
				},
				"dnsPolicy": "Default",
			},
		},
	}
}

// skipReason is the reason recorded for a phase skipped because condition
// evaluated to false.
func skipReason(condition string) string {
	return fmt.Sprintf("if=`%s` evaluated to false.", condition)
}

func assertPhase(c *check.C, p crv1alpha1.Phase, name string, state crv1alpha1.State, reason string) {
	c.Assert(p.Name, check.Equals, name)
	c.Assert(p.State, check.Equals, state, check.Commentf("phase %s", name))
	c.Assert(p.Reason, check.Equals, reason, check.Commentf("phase %s", name))
}

func phasesOf(c *check.C, as *crv1alpha1.ActionSet, n int) []crv1alpha1.Phase {
	c.Assert(as.Status.Actions, check.HasLen, 1)
	phases := as.Status.Actions[0].Phases
	c.Assert(phases, check.HasLen, n)
	return phases
}

// TestConditionalPhaseExecution runs one phase whose condition is true and
// one whose condition is false. The skipped phase would fail if it were run.
func (s *E2ESuite) TestConditionalPhaseExecution(c *check.C) {
	ctx, can := context.WithTimeout(context.Background(), conditionalPhaseTestTimeout)
	defer can()
	t := s.setUpConditionalPhaseTarget(ctx, c)

	const skipCondition = `{{ eq .Deployment.Namespace "no-such-namespace" }}`
	as := s.runConditionalActionSet(ctx, c, t, []crv1alpha1.BlueprintPhase{
		execPhase("executed-phase", `{{ gt (len .Deployment.Pods) 0 }}`, "echo", "hello"),
		execPhase("skipped-phase", skipCondition, "false"),
	}, nil)

	c.Assert(as.Status.State, check.Equals, crv1alpha1.StateComplete)
	phases := phasesOf(c, as, 2)
	assertPhase(c, phases[0], "executed-phase", crv1alpha1.StateComplete, "")
	assertPhase(c, phases[1], "skipped-phase", crv1alpha1.StateSkipped, skipReason(skipCondition))

	log.Print("Completed E2E TestConditionalPhaseExecution")
}

// TestConditionalPhaseOnOptions evaluates the same condition against different
// ActionSet options.
func (s *E2ESuite) TestConditionalPhaseOnOptions(c *check.C) {
	ctx, can := context.WithTimeout(context.Background(), conditionalPhaseTestTimeout)
	defer can()
	t := s.setUpConditionalPhaseTarget(ctx, c)

	const condition = `{{ eq .Options.mode "full" }}`
	for _, tc := range []struct {
		mode   string
		state  crv1alpha1.State
		reason string
	}{
		{mode: "full", state: crv1alpha1.StateComplete, reason: ""},
		{mode: "incremental", state: crv1alpha1.StateSkipped, reason: skipReason(condition)},
	} {
		as := s.runConditionalActionSet(ctx, c, t, []crv1alpha1.BlueprintPhase{
			execPhase("full-only", condition, "echo", "full backup"),
		}, map[string]string{"mode": tc.mode})

		c.Assert(as.Status.State, check.Equals, crv1alpha1.StateComplete, check.Commentf("mode %s", tc.mode))
		assertPhase(c, phasesOf(c, as, 1)[0], "full-only", tc.state, tc.reason)
	}

	log.Print("Completed E2E TestConditionalPhaseOnOptions")
}

// TestConditionalPhaseAllSkipped checks that an ActionSet whose phases are
// all skipped still completes.
func (s *E2ESuite) TestConditionalPhaseAllSkipped(c *check.C) {
	ctx, can := context.WithTimeout(context.Background(), conditionalPhaseTestTimeout)
	defer can()
	t := s.setUpConditionalPhaseTarget(ctx, c)

	const (
		firstCondition  = `{{ eq 1 2 }}`
		secondCondition = `{{ eq .Deployment.Namespace "no-such-namespace" }}`
	)
	as := s.runConditionalActionSet(ctx, c, t, []crv1alpha1.BlueprintPhase{
		execPhase("first", firstCondition, "false"),
		execPhase("second", secondCondition, "false"),
	}, nil)

	c.Assert(as.Status.State, check.Equals, crv1alpha1.StateComplete)
	phases := phasesOf(c, as, 2)
	assertPhase(c, phases[0], "first", crv1alpha1.StateSkipped, skipReason(firstCondition))
	assertPhase(c, phases[1], "second", crv1alpha1.StateSkipped, skipReason(secondCondition))

	log.Print("Completed E2E TestConditionalPhaseAllSkipped")
}

// TestConditionalPhaseSkippedInMiddle checks that phases after a skipped
// phase still run.
func (s *E2ESuite) TestConditionalPhaseSkippedInMiddle(c *check.C) {
	ctx, can := context.WithTimeout(context.Background(), conditionalPhaseTestTimeout)
	defer can()
	t := s.setUpConditionalPhaseTarget(ctx, c)

	const skipCondition = `{{ eq .Deployment.Namespace "no-such-namespace" }}`
	as := s.runConditionalActionSet(ctx, c, t, []crv1alpha1.BlueprintPhase{
		execPhase("before", "", "echo", "before"),
		execPhase("skipped", skipCondition, "false"),
		execPhase("after", "", "echo", "after"),
	}, nil)

	c.Assert(as.Status.State, check.Equals, crv1alpha1.StateComplete)
	phases := phasesOf(c, as, 3)
	assertPhase(c, phases[0], "before", crv1alpha1.StateComplete, "")
	assertPhase(c, phases[1], "skipped", crv1alpha1.StateSkipped, skipReason(skipCondition))
	assertPhase(c, phases[2], "after", crv1alpha1.StateComplete, "")

	log.Print("Completed E2E TestConditionalPhaseSkippedInMiddle")
}

// TestConditionalPhaseOnPreviousPhaseOutput evaluates a condition against the
// output of an earlier phase.
func (s *E2ESuite) TestConditionalPhaseOnPreviousPhaseOutput(c *check.C) {
	ctx, can := context.WithTimeout(context.Background(), conditionalPhaseTestTimeout)
	defer can()
	t := s.setUpConditionalPhaseTarget(ctx, c)

	const condition = `{{ eq .Phases.setFlag.Output.flag "true" }}`
	for _, tc := range []struct {
		flag   string
		state  crv1alpha1.State
		reason string
	}{
		{flag: "true", state: crv1alpha1.StateComplete, reason: ""},
		{flag: "false", state: crv1alpha1.StateSkipped, reason: skipReason(condition)},
	} {
		as := s.runConditionalActionSet(ctx, c, t, []crv1alpha1.BlueprintPhase{
			outputPhase("setFlag", "", "flag", tc.flag),
			execPhase("dependent", condition, "echo", "flag was true"),
		}, nil)

		c.Assert(as.Status.State, check.Equals, crv1alpha1.StateComplete, check.Commentf("flag %s", tc.flag))
		phases := phasesOf(c, as, 2)
		assertPhase(c, phases[0], "setFlag", crv1alpha1.StateComplete, "")
		assertPhase(c, phases[1], "dependent", tc.state, tc.reason)
	}

	log.Print("Completed E2E TestConditionalPhaseOnPreviousPhaseOutput")
}

// TestConditionalPhaseProgress checks that skipped phases count as done
// when computing progress.
func (s *E2ESuite) TestConditionalPhaseProgress(c *check.C) {
	ctx, can := context.WithTimeout(context.Background(), conditionalPhaseTestTimeout)
	defer can()
	t := s.setUpConditionalPhaseTarget(ctx, c)

	const skipCondition = `{{ eq .Deployment.Namespace "no-such-namespace" }}`
	as := s.runConditionalActionSet(ctx, c, t, []crv1alpha1.BlueprintPhase{
		execPhase("run-1", "", "echo", "one"),
		execPhase("skip-1", skipCondition, "false"),
		execPhase("run-2", "", "echo", "two"),
		execPhase("skip-2", skipCondition, "false"),
	}, nil)

	c.Assert(as.Status.State, check.Equals, crv1alpha1.StateComplete)
	c.Assert(as.Status.Progress.PercentCompleted, check.Equals, "100")
	phases := phasesOf(c, as, 4)
	for _, i := range []int{1, 3} {
		assertPhase(c, phases[i], phases[i].Name, crv1alpha1.StateSkipped, skipReason(skipCondition))
		c.Assert(phases[i].Progress.ProgressPercent, check.Equals, "100", check.Commentf("phase %s", phases[i].Name))
	}

	log.Print("Completed E2E TestConditionalPhaseProgress")
}

// TestConditionalPhaseInvalidCondition checks that a condition that can't be
// evaluated to true or false fails the phase and the ActionSet, instead of
// skipping the phase.
func (s *E2ESuite) TestConditionalPhaseInvalidCondition(c *check.C) {
	ctx, can := context.WithTimeout(context.Background(), conditionalPhaseTestTimeout)
	defer can()
	t := s.setUpConditionalPhaseTarget(ctx, c)

	for _, tc := range []struct {
		name          string
		condition     string
		expectedError error
	}{
		{
			name:          "non-boolean result",
			condition:     `{{ "test" }}`,
			expectedError: kanister.ErrPhaseExpressionNotConditional,
		},
		{
			name:          "execution error",
			condition:     `{{ .Invalid.Option }}`,
			expectedError: kanister.ErrExecutePhaseCondition,
		},
		{
			name:          "syntax error",
			condition:     `{{`,
			expectedError: kanister.ErrParsePhaseCondition,
		},
	} {
		as := s.runConditionalActionSet(ctx, c, t, []crv1alpha1.BlueprintPhase{
			execPhase("invalid", tc.condition, "echo", "hello"),
			execPhase("not-reached", "", "echo", "hello"),
		}, nil)

		c.Assert(as.Status.State, check.Equals, crv1alpha1.StateFailed, check.Commentf(tc.name))
		c.Assert(strings.Contains(as.Status.Error.Message, tc.expectedError.Error()), check.Equals, true,
			check.Commentf("%s: error message %q does not contain %q", tc.name, as.Status.Error.Message, tc.expectedError.Error()))
		phases := phasesOf(c, as, 2)
		assertPhase(c, phases[0], "invalid", crv1alpha1.StateFailed, "")
		assertPhase(c, phases[1], "not-reached", crv1alpha1.StatePending, "")
	}

	log.Print("Completed E2E TestConditionalPhaseInvalidCondition")
}

// TestConditionalPhaseSkippedThenFailed checks that a skipped phase keeps its
// state when a later phase fails.
func (s *E2ESuite) TestConditionalPhaseSkippedThenFailed(c *check.C) {
	ctx, can := context.WithTimeout(context.Background(), conditionalPhaseTestTimeout)
	defer can()
	t := s.setUpConditionalPhaseTarget(ctx, c)

	const skipCondition = `{{ eq .Deployment.Namespace "no-such-namespace" }}`
	as := s.runConditionalActionSet(ctx, c, t, []crv1alpha1.BlueprintPhase{
		execPhase("skipped", skipCondition, "false"),
		execPhase("failing", "", "false"),
	}, nil)

	c.Assert(as.Status.State, check.Equals, crv1alpha1.StateFailed)
	phases := phasesOf(c, as, 2)
	assertPhase(c, phases[0], "skipped", crv1alpha1.StateSkipped, skipReason(skipCondition))
	assertPhase(c, phases[1], "failing", crv1alpha1.StateFailed, "")

	log.Print("Completed E2E TestConditionalPhaseSkippedThenFailed")
}

// TestConditionalPhaseChildArtifactInheritance checks that a skipped phase in
// the parent ActionSet doesn't stop a child ActionSet from consuming the
// artifact the parent produced.
func (s *E2ESuite) TestConditionalPhaseChildArtifactInheritance(c *check.C) {
	ctx, can := context.WithTimeout(context.Background(), conditionalPhaseTestTimeout)
	defer can()
	t := s.setUpConditionalPhaseTarget(ctx, c)

	const (
		skipCondition = `{{ eq .Options.mode "full" }}`
		artifactPath  = "/backup/snapshot-1"
	)
	bpName := s.createConditionalBlueprint(ctx, c, map[string]*crv1alpha1.BlueprintAction{
		"backup": {
			Kind: "Deployment",
			OutputArtifacts: map[string]crv1alpha1.Artifact{
				"cloudObject": {
					KeyValue: map[string]string{
						"path": "/backup/{{ .Phases.produce.Output.path }}",
					},
				},
			},
			Phases: []crv1alpha1.BlueprintPhase{
				execPhase("skipped", skipCondition, "false"),
				outputPhase("produce", "", "path", "snapshot-1"),
			},
		},
		"restore": {
			Kind:               "Deployment",
			InputArtifactNames: []string{"cloudObject"},
			Phases: []crv1alpha1.BlueprintPhase{
				execPhase("consume", "", "echo", "{{ .ArtifactsIn.cloudObject.KeyValue.path }}"),
			},
		},
	})

	parent := s.runConditionalAction(ctx, c, t, bpName, "backup", map[string]string{"mode": "incremental"})
	c.Assert(parent.Status.State, check.Equals, crv1alpha1.StateComplete)
	parentPhases := phasesOf(c, parent, 2)
	assertPhase(c, parentPhases[0], "skipped", crv1alpha1.StateSkipped, skipReason(skipCondition))
	assertPhase(c, parentPhases[1], "produce", crv1alpha1.StateComplete, "")
	c.Assert(parent.Status.Actions[0].Artifacts["cloudObject"].KeyValue["path"], check.Equals, artifactPath)

	child := s.runChildActionSet(ctx, c, parent, "restore", nil)
	c.Assert(child.Spec.Actions[0].Artifacts["cloudObject"].KeyValue["path"], check.Equals, artifactPath)
	c.Assert(child.Status.State, check.Equals, crv1alpha1.StateComplete)
	assertPhase(c, phasesOf(c, child, 1)[0], "consume", crv1alpha1.StateComplete, "")

	log.Print("Completed E2E TestConditionalPhaseChildArtifactInheritance")
}

// TestConditionalPhaseChildReevaluation checks that a child ActionSet doesn't
// inherit the parent's skip decisions: it evaluates every condition again with
// the options it inherits or overrides.
func (s *E2ESuite) TestConditionalPhaseChildReevaluation(c *check.C) {
	ctx, can := context.WithTimeout(context.Background(), conditionalPhaseTestTimeout)
	defer can()
	t := s.setUpConditionalPhaseTarget(ctx, c)

	const condition = `{{ eq .Options.mode "full" }}`
	bpName := s.createConditionalBlueprint(ctx, c, map[string]*crv1alpha1.BlueprintAction{
		"run": {
			Kind: "Deployment",
			Phases: []crv1alpha1.BlueprintPhase{
				execPhase("optional", condition, "echo", "full"),
			},
		},
	})

	parent := s.runConditionalAction(ctx, c, t, bpName, "run", map[string]string{"mode": "incremental"})
	c.Assert(parent.Status.State, check.Equals, crv1alpha1.StateComplete)
	assertPhase(c, phasesOf(c, parent, 1)[0], "optional", crv1alpha1.StateSkipped, skipReason(condition))

	// Inherited options keep the condition false, so the phase is skipped again.
	inherited := s.runChildActionSet(ctx, c, parent, "run", nil)
	c.Assert(inherited.Spec.Actions[0].Options["mode"], check.Equals, "incremental")
	c.Assert(inherited.Status.State, check.Equals, crv1alpha1.StateComplete)
	assertPhase(c, phasesOf(c, inherited, 1)[0], "optional", crv1alpha1.StateSkipped, skipReason(condition))

	// An overriding option makes the condition true, so the phase runs.
	overridden := s.runChildActionSet(ctx, c, parent, "run", map[string]string{"mode": "full"})
	c.Assert(overridden.Spec.Actions[0].Options["mode"], check.Equals, "full")
	c.Assert(overridden.Status.State, check.Equals, crv1alpha1.StateComplete)
	assertPhase(c, phasesOf(c, overridden, 1)[0], "optional", crv1alpha1.StateComplete, "")

	log.Print("Completed E2E TestConditionalPhaseChildReevaluation")
}
