package progress

import (
	"context"
	"fmt"
	"testing"

	"github.com/kanisterio/kanister/pkg/client/clientset/versioned"
	"github.com/kanisterio/kanister/pkg/client/clientset/versioned/fake"
	. "gopkg.in/check.v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testBlueprint = "test-blueprint-progress"
	testActionset = "test-actionset-progress"
	testNamespace = "default"
)

func Test(t *testing.T) {
	TestingT(t)
}

type TestSuiteSinglePhase struct {
	blueprint *crv1alpha1.Blueprint
	actionSet *crv1alpha1.ActionSet
	clientset *fake.Clientset
}

var _ = Suite(&TestSuiteSinglePhase{})

func (s *TestSuiteSinglePhase) SetUpTest(c *C) {
	mockBlueprint := &crv1alpha1.Blueprint{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      testBlueprint,
			Namespace: testNamespace,
		},
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"action-01": {
				Name: "action-01",
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Name: "echo-hello-0-0",
						Func: "echo-hello-func",
					},
				},
			},
		},
	}

	mockActionSet := &crv1alpha1.ActionSet{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      testActionset,
			Namespace: testNamespace,
		},
		Spec: &crv1alpha1.ActionSetSpec{
			Actions: []crv1alpha1.ActionSpec{
				{
					Name: "action-01",
					Object: crv1alpha1.ObjectReference{
						APIVersion: "v1",
						Group:      "",
						Resource:   "Namespace",
					},
					Blueprint: testBlueprint,
				},
			},
		},
		Status: &crv1alpha1.ActionSetStatus{
			State: crv1alpha1.StateRunning,
			Actions: []crv1alpha1.ActionStatus{
				{
					Name: "action-01",
					Phases: []crv1alpha1.Phase{
						{
							Name:  "echo-hello-0-0",
							State: crv1alpha1.StatePending,
						},
					},
				},
			},
		},
	}

	s.clientset = fake.NewSimpleClientset()
	err := s.createFixtures(mockBlueprint, mockActionSet)
	c.Assert(err, IsNil)
}

func (s *TestSuiteSinglePhase) TearDownTest(c *C) {
	blueprintErr := s.clientset.CrV1alpha1().Blueprints(s.blueprint.GetNamespace()).Delete(
		context.Background(),
		s.blueprint.GetName(),
		metav1.DeleteOptions{})
	c.Assert(blueprintErr, IsNil)

	actionSetErr := s.clientset.CrV1alpha1().ActionSets(s.actionSet.GetNamespace()).Delete(
		context.Background(),
		s.actionSet.GetName(),
		metav1.DeleteOptions{})
	c.Assert(actionSetErr, IsNil)
}

func (s *TestSuiteSinglePhase) createFixtures(blueprint *crv1alpha1.Blueprint, actionSet *crv1alpha1.ActionSet) error {
	createdBlueprint, err := s.clientset.CrV1alpha1().Blueprints(blueprint.GetNamespace()).Create(
		context.Background(),
		blueprint,
		metav1.CreateOptions{})
	if err != nil {
		return err
	}
	s.blueprint = createdBlueprint

	createdActionSet, err := s.clientset.CrV1alpha1().ActionSets(actionSet.GetNamespace()).Create(
		context.Background(),
		actionSet,
		metav1.CreateOptions{})
	if err != nil {
		return err
	}
	s.actionSet = createdActionSet

	return nil
}

func (s *TestSuiteSinglePhase) TestUpdateActionPhaseProgress(c *C) {
	var testCases = []struct {
		indexAction           int
		indexPhase            int
		phaseState            crv1alpha1.State
		phasePercent          string
		expectedPhasePercent  string
		expectedActionPercent string
	}{
		{
			phaseState:            crv1alpha1.StatePending,
			expectedPhasePercent:  "",
			expectedActionPercent: "",
		},
		{
			phaseState:            crv1alpha1.StateRunning,
			phasePercent:          "20",
			expectedPhasePercent:  "20",
			expectedActionPercent: "20",
		},
		{
			phaseState:            crv1alpha1.StateRunning,
			phasePercent:          "50",
			expectedPhasePercent:  "50",
			expectedActionPercent: "50",
		},
		{
			phaseState:            crv1alpha1.StateFailed,
			phasePercent:          "50",
			expectedPhasePercent:  "",
			expectedActionPercent: "",
		},
		{
			phaseState:            crv1alpha1.StateComplete,
			phasePercent:          CompletedPercent,
			expectedPhasePercent:  CompletedPercent,
			expectedActionPercent: CompletedPercent,
		},
	}
	for id, tc := range testCases {
		assertActionProgress(
			c,
			s.clientset,
			s.actionSet,
			tc.indexAction,
			tc.indexPhase,
			tc.phaseState,
			tc.phasePercent,
			tc.expectedPhasePercent,
			tc.expectedActionPercent,
			id)
	}
}

func assertActionProgress(
	c *C,
	clientset versioned.Interface,
	actionSet *crv1alpha1.ActionSet,
	indexAction int,
	indexPhase int,
	phaseState crv1alpha1.State,
	phasePercent string,
	expectedPhasePercent string,
	expectedActionPercent string,
	testCaseID int) {
	now := metav1.Now()
	actionSet.Status.Actions[indexAction].Phases[indexPhase].State = phaseState
	updated, err := clientset.CrV1alpha1().ActionSets(actionSet.GetNamespace()).Update(context.Background(), actionSet, metav1.UpdateOptions{})
	c.Assert(err, IsNil)
	phaseName := fmt.Sprintf("echo-hello-%d-%d", indexAction, indexPhase)
	err1 := updateActionSetStatus(context.Background(), clientset, actionSet, phaseName, crv1alpha1.PhaseProgress{ProgressPercent: phasePercent, LastTransitionTime: &now})
	c.Assert(err1, IsNil, Commentf("test case #: %d", testCaseID))
	actual, err := clientset.CrV1alpha1().ActionSets(actionSet.GetNamespace()).Get(context.Background(), updated.GetName(), metav1.GetOptions{})
	c.Assert(err, IsNil)
	// Check phase progress percent
	c.Assert(actual.Status.Actions[indexAction].Phases[indexPhase].Progress.ProgressPercent, Equals, expectedPhasePercent, Commentf("test case #: %d", testCaseID))
	// Check action progress percent
	c.Assert(actual.Status.Progress.PercentCompleted, Equals, expectedActionPercent, Commentf("test case #: %d", testCaseID))
	if phaseState != crv1alpha1.StateFailed &&
		phaseState != crv1alpha1.StatePending {
		c.Assert(actual.Status.Actions[indexAction].Phases[indexPhase].Progress.LastTransitionTime, NotNil)
		c.Assert(*actual.Status.Actions[indexAction].Phases[indexPhase].Progress.LastTransitionTime, Equals, now, Commentf("test case #: %d", testCaseID))
	}
}
