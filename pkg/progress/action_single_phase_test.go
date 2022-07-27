package progress

import (
	"context"
	"testing"
	"time"

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
						Name: "echo-hello",
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
							Name:  "echo-hello",
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

func (s *TestSuiteSinglePhase) TestUpdateActionsProgress(c *C) {
	now := time.Now()
	var testCases = []struct {
		indexAction int
		indexPhase  int
		phaseState  crv1alpha1.State
		expected    string
	}{
		{
			phaseState: crv1alpha1.StatePending,
			expected:   progressPercentStarted,
		},
		{
			phaseState: crv1alpha1.StateRunning,
			expected:   progressPercentStarted,
		},
		{
			phaseState: crv1alpha1.StateFailed,
			expected:   progressPercentStarted,
		},
		{
			phaseState: crv1alpha1.StateComplete,
			expected:   progressPercentCompleted,
		},
	}

	for id, tc := range testCases {
		assertActionProgress(
			c,
			s.clientset,
			s.actionSet,
			now,
			tc.indexAction,
			tc.indexPhase,
			tc.phaseState,
			tc.expected,
			id)
	}
}

func assertActionProgress(
	c *C,
	clientset versioned.Interface,
	actionSet *crv1alpha1.ActionSet,
	now time.Time,
	indexAction int,
	indexPhase int,
	newState crv1alpha1.State,
	expectedPercentCompleted string,
	testCaseID int) {
	// update the phase state
	actionSet.Status.Actions[indexAction].Phases[indexPhase].State = newState
	updated, err := clientset.CrV1alpha1().ActionSets(actionSet.GetNamespace()).Update(context.Background(), actionSet, metav1.UpdateOptions{})
	c.Assert(err, IsNil)

	// calculate and update the progress so that it reflects the state change
	phaseWeights, totalWeight, err := calculatePhaseWeights(context.Background(), actionSet.GetName(), actionSet.GetNamespace(), clientset)
	c.Assert(err, IsNil)

	updateErr := updateActionsProgress(context.Background(), clientset, updated, phaseWeights, totalWeight, now)
	c.Assert(updateErr, IsNil)

	// retrieve the latest actionSet resource to confirm its progress data
	actual, err := clientset.CrV1alpha1().ActionSets(actionSet.GetNamespace()).Get(context.Background(), updated.GetName(), metav1.GetOptions{})
	c.Assert(err, IsNil, Commentf("test case #: %d", testCaseID))
	c.Assert(actual.Status.Progress.PercentCompleted, Equals, expectedPercentCompleted, Commentf("test case #: %d", testCaseID))
	c.Assert(actual.Status.Progress.LastTransitionTime.Time, Equals, now, Commentf("test case #: %d", testCaseID))
}
