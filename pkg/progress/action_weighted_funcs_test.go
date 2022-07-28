package progress

import (
	"context"
	"time"

	"github.com/kanisterio/kanister/pkg/client/clientset/versioned/fake"
	. "gopkg.in/check.v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TestSuiteWeightedFuncs struct {
	blueprint *crv1alpha1.Blueprint
	actionSet *crv1alpha1.ActionSet
	clientset *fake.Clientset
}

var _ = Suite(&TestSuiteWeightedFuncs{})

func (s *TestSuiteWeightedFuncs) SetUpTest(c *C) {
	mockBlueprint := &crv1alpha1.Blueprint{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      testBlueprint,
			Namespace: testNamespace,
		},
		Actions: map[string]*crv1alpha1.BlueprintAction{
			"action-weighted": {
				Name: "action-weighted",
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Name: "echo-00",
						Func: "BackupData",
					},
					{
						Name: "echo-01",
						Func: "CopyVolumeData",
					},
				},
			},
			"action-normal": {
				Name: "action-normal",
				Phases: []crv1alpha1.BlueprintPhase{
					{
						Name: "echo-02",
						Func: "echo-hello-func",
					},
					{
						Name: "echo-03",
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
					Name: "action-weighted",
					Object: crv1alpha1.ObjectReference{
						APIVersion: "v1",
						Group:      "",
						Resource:   "Namespace",
					},
					Blueprint: testBlueprint,
				},
				{
					Name: "action-normal",
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
					Name: "action-weighted",
					Phases: []crv1alpha1.Phase{
						{
							Name:  "echo-00",
							State: crv1alpha1.StatePending,
						},
						{
							Name:  "echo-01",
							State: crv1alpha1.StatePending,
						},
					},
				},
				{
					Name: "action-normal",
					Phases: []crv1alpha1.Phase{
						{
							Name:  "echo-02",
							State: crv1alpha1.StatePending,
						},
						{
							Name:  "echo-03",
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

func (s *TestSuiteWeightedFuncs) TearDownTest(c *C) {
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

func (s *TestSuiteWeightedFuncs) createFixtures(blueprint *crv1alpha1.Blueprint, actionSet *crv1alpha1.ActionSet) error {
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

func (s *TestSuiteWeightedFuncs) TestUpdateActionsProgress(c *C) {
	now := time.Now()
	var testCases = []struct {
		indexAction int
		indexPhase  int
		phaseState  crv1alpha1.State
		expected    string
	}{
		{
			indexAction: 0,
			indexPhase:  0,
			phaseState:  crv1alpha1.StatePending,
			expected:    progressPercentStarted,
		},
		{
			indexAction: 0,
			indexPhase:  0,
			phaseState:  crv1alpha1.StateRunning,
			expected:    progressPercentStarted,
		},
		{
			indexAction: 0,
			indexPhase:  0,
			phaseState:  crv1alpha1.StateComplete,
			expected:    "33.33",
		},
		{
			indexAction: 0,
			indexPhase:  1,
			phaseState:  crv1alpha1.StatePending,
			expected:    "33.33",
		},
		{
			indexAction: 0,
			indexPhase:  1,
			phaseState:  crv1alpha1.StateRunning,
			expected:    "33.33",
		},
		{
			indexAction: 0,
			indexPhase:  1,
			phaseState:  crv1alpha1.StateComplete,
			expected:    "66.67",
		},
		{
			indexAction: 1,
			indexPhase:  0,
			phaseState:  crv1alpha1.StatePending,
			expected:    "66.67",
		},
		{
			indexAction: 1,
			indexPhase:  0,
			phaseState:  crv1alpha1.StateRunning,
			expected:    "66.67",
		},
		{
			indexAction: 1,
			indexPhase:  0,
			phaseState:  crv1alpha1.StateComplete,
			expected:    "83.33",
		},
		{
			indexAction: 1,
			indexPhase:  1,
			phaseState:  crv1alpha1.StatePending,
			expected:    "83.33",
		},
		{
			indexAction: 1,
			indexPhase:  1,
			phaseState:  crv1alpha1.StateRunning,
			expected:    "83.33",
		},
		{
			indexAction: 1,
			indexPhase:  1,
			phaseState:  crv1alpha1.StateComplete,
			expected:    progressPercentCompleted,
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
