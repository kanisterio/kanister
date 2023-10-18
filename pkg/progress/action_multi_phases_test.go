package progress

import (
	"context"

	"github.com/kanisterio/kanister/pkg/client/clientset/versioned/fake"
	. "gopkg.in/check.v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TestSuiteMultiPhases struct {
	blueprint *crv1alpha1.Blueprint
	actionSet *crv1alpha1.ActionSet
	clientset *fake.Clientset
}

var _ = Suite(&TestSuiteMultiPhases{})

func (s *TestSuiteMultiPhases) SetUpTest(c *C) {
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
					{
						Name: "echo-hello-0-1",
						Func: "echo-world-func",
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
						{
							Name:  "echo-hello-0-1",
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

func (s *TestSuiteMultiPhases) TearDownTest(c *C) {
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

func (s *TestSuiteMultiPhases) createFixtures(blueprint *crv1alpha1.Blueprint, actionSet *crv1alpha1.ActionSet) error {
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

func (s *TestSuiteMultiPhases) TestUpdateActionsProgress(c *C) {
	var testCases = []struct {
		indexAction           int
		indexPhase            int
		phaseState            crv1alpha1.State
		phasePercent          string
		expectedPhasePercent  string
		expectedActionPercent string
	}{
		{
			indexAction:           0,
			indexPhase:            0,
			phaseState:            crv1alpha1.StatePending,
			phasePercent:          "0",
			expectedPhasePercent:  "",
			expectedActionPercent: "",
		},
		{
			indexAction:           0,
			indexPhase:            0,
			phaseState:            crv1alpha1.StateRunning,
			phasePercent:          "30",
			expectedPhasePercent:  "30",
			expectedActionPercent: "15",
		},
		{
			indexAction:           0,
			indexPhase:            0,
			phaseState:            crv1alpha1.StateComplete,
			phasePercent:          CompletedPercent,
			expectedPhasePercent:  CompletedPercent,
			expectedActionPercent: "50",
		},
		{
			indexAction:           0,
			indexPhase:            1,
			phaseState:            crv1alpha1.StatePending,
			phasePercent:          "0",
			expectedPhasePercent:  "",
			expectedActionPercent: "50", // stays at 50% because 1st action is done
		},
		{
			indexAction:           0,
			indexPhase:            1,
			phaseState:            crv1alpha1.StateRunning,
			phasePercent:          "60",
			expectedPhasePercent:  "60",
			expectedActionPercent: "80", // Averaging out action progress 100(phase-1)+60(phase-2)/2(total phases)
		},
		{
			indexAction:           0,
			indexPhase:            1,
			phaseState:            crv1alpha1.StateComplete,
			phasePercent:          CompletedPercent,
			expectedPhasePercent:  CompletedPercent,
			expectedActionPercent: CompletedPercent,
		},
	}

	for id, tc := range testCases {
		// Get latest rev of actionset resource
		as, err := s.clientset.CrV1alpha1().ActionSets(s.actionSet.GetNamespace()).Get(context.Background(), s.actionSet.GetName(), metav1.GetOptions{})
		c.Assert(err, IsNil)
		assertActionProgress(
			c,
			s.clientset,
			as,
			tc.indexAction,
			tc.indexPhase,
			tc.phaseState,
			tc.phasePercent,
			tc.expectedPhasePercent,
			tc.expectedActionPercent,
			id)
	}
}

func (s *TestSuiteMultiPhases) TestUpdateActionsProgressWithFailures(c *C) {
	var testCases = []struct {
		indexAction           int
		indexPhase            int
		phaseState            crv1alpha1.State
		phasePercent          string
		expectedPhasePercent  string
		expectedActionPercent string
	}{
		{
			indexAction:           0,
			indexPhase:            0,
			phaseState:            crv1alpha1.StateComplete,
			phasePercent:          CompletedPercent,
			expectedPhasePercent:  CompletedPercent,
			expectedActionPercent: "50",
		},
		{
			indexAction:           0,
			indexPhase:            1,
			phaseState:            crv1alpha1.StateFailed,
			phasePercent:          "30",
			expectedPhasePercent:  "",
			expectedActionPercent: "50", // stays at 50% because 1st action is done
		},
	}

	for id, tc := range testCases {
		// Get latest rev of actionset resource
		as, err := s.clientset.CrV1alpha1().ActionSets(s.actionSet.GetNamespace()).Get(context.Background(), s.actionSet.GetName(), metav1.GetOptions{})
		c.Assert(err, IsNil)
		assertActionProgress(
			c,
			s.clientset,
			as,
			tc.indexAction,
			tc.indexPhase,
			tc.phaseState,
			tc.phasePercent,
			tc.expectedPhasePercent,
			tc.expectedActionPercent,
			id)
	}
}
